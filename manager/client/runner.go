package client

import (
	"context"
	"errors"
	"sync"
	"time"

	shared "github.com/figment-networks/cosmos-indexer/structs"
)

type Progresser interface {
	Report(done shared.HeightRange, duration time.Duration, err []error, finished bool)
}

type Runner struct {
	runs map[NetworkVersion][]*Run
	in   chan RunReq
	out  chan chan ReqOut
}

func NewRunner() *Runner {
	return &Runner{
		in:   make(chan RunReq, 2),
		runs: make(map[NetworkVersion][]*Run),
		out:  make(chan chan ReqOut, 2),
	}
}

type ReqOut struct {
	Run []Run `json:"runs"`
	Err error `json:"error"`
}

type RunResp struct {
	IsNew bool
	Run   *Run
	Err   error
}

type RunReq struct {
	Kind        string //Type
	NV          NetworkVersion
	HeightRange shared.HeightRange

	Force bool
	Resp  chan RunResp
}

func (r *Runner) StartProcess(nv NetworkVersion, heightRange shared.HeightRange, force bool) (bool, *Run, error) {
	resp := make(chan RunResp, 1)
	defer close(resp)
	r.in <- RunReq{Kind: "start", NV: nv, HeightRange: heightRange, Force: force, Resp: resp}
	response := <-resp
	return response.IsNew, response.Run, response.Err
}

func (r *Runner) StopProcess(nv NetworkVersion, heightRange shared.HeightRange) error {
	resp := make(chan RunResp, 1)
	defer close(resp)
	r.in <- RunReq{Kind: "stop", NV: nv, HeightRange: heightRange, Resp: resp}
	response := <-resp
	return response.Err
}

func (r *Runner) GetRunning() ReqOut {
	resp := make(chan ReqOut, 1)
	defer close(resp)
	r.out <- resp
	return <-resp
}

func (r *Runner) Run() {
	ctx := context.Background()
	tckr := time.NewTicker(time.Hour)
	for {
		select {
		case <-tckr.C:
			r.cleanup()
		case out := <-r.out:
			ro := ReqOut{}
			for _, runs := range r.runs {
				for _, run := range runs {
					ro.Run = append(ro.Run, Run{NV: run.NV,
						HeightRange:      run.HeightRange,
						Success:          run.Success,
						FinishTime:       run.FinishTime,
						Finished:         run.Finished,
						Progress:         run.Progress,
						LastProgressTime: run.LastProgressTime})
				}
			}
			out <- ro
		case rReq := <-r.in:
			if rReq.Kind == "start" {
				currentlyRunning, ok := r.checkExisting(rReq)
				if ok || rReq.Force {
					nCtx, cancel := context.WithCancel(ctx)

					run := NewRun(nCtx, rReq.NV, rReq.HeightRange)
					run.Cancel = cancel
					k, ok := r.runs[rReq.NV]
					if !ok {
						k = []*Run{}
					}
					r.runs[rReq.NV] = append(k, run)
					rReq.Resp <- RunResp{IsNew: true, Run: run, Err: nil}
					continue
				}

				if !ok && !rReq.Force {
					rReq.Resp <- RunResp{IsNew: false, Run: currentlyRunning, Err: nil}
					continue
				}

				rReq.Resp <- RunResp{IsNew: false, Run: nil, Err: errors.New("error occured")}
			} else if rReq.Kind == "stop" {
				currentlyRunning, ok := r.checkExisting(rReq)
				if ok {
					currentlyRunning.Stop()
				}
				rReq.Resp <- RunResp{IsNew: false, Run: nil, Err: nil}
			}

			rReq.Resp <- RunResp{IsNew: false, Run: nil, Err: errors.New("unknown request kind")}
		}
	}
}

func (r *Runner) cleanup() {
	for k, runs := range r.runs {
		newRuns := []*Run{}
		for _, r := range runs {
			if !(r.Finished && time.Since(r.FinishTime) > time.Hour*2) {
				rt := r
				newRuns = append(newRuns, rt)
			}
		}

		if len(newRuns) > 0 {
			r.runs[k] = newRuns
		} else {
			delete(r.runs, k)
		}

	}
}

func (r *Runner) checkExisting(req RunReq) (currentlyRunning *Run, ok bool) {
	current, ok := r.runs[req.NV]
	if !ok {
		return nil, true
	}

	for _, run := range current {
		// totally inclusive
		if (req.HeightRange.StartHeight >= run.HeightRange.StartHeight && req.HeightRange.StartHeight <= run.HeightRange.EndHeight) &&
			(req.HeightRange.EndHeight <= run.HeightRange.EndHeight && req.HeightRange.EndHeight >= run.HeightRange.StartHeight) {
			if !run.Finished {
				currentlyRunning = run
			}
			return currentlyRunning, false
		}

		// TODO(lukanus): support other ranges (split), this is just optimisation

	}
	return nil, true
}

type Run struct {
	sync.Mutex `json:"-"`

	NV          NetworkVersion     `json:"nv"`
	HeightRange shared.HeightRange `json:"height_range"`

	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	Progress []RunProgress `json:"run_progress"`

	Success bool `json:"success"`

	Finished         bool      `json:"finished"`
	FinishTime       time.Time `json:"finished_time"`
	LastProgressTime time.Time `json:"last_progress_time"`
}

func NewRun(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange) *Run {

	return &Run{
		Ctx:         ctx,
		NV:          nv,
		HeightRange: heightRange,
		Progress:    []RunProgress{},
	}
}

func (r *Run) Report(done shared.HeightRange, duration time.Duration, err []error, finished bool) {
	r.Lock()
	defer r.Unlock()

	r.LastProgressTime = time.Now()
	r.Progress = append(r.Progress, RunProgress{Done: done, D: duration, T: time.Now(), Errors: err})

	if finished {
		r.Success = true
		for _, p := range r.Progress {
			if len(p.Errors) > 0 {
				r.Success = false
			}
		}

		r.Finished = true
		r.FinishTime = time.Now()
	}
}

func (r *Run) Stop() {
	r.Lock()
	defer r.Unlock()

	r.Cancel()
	r.Finished = true
	r.FinishTime = time.Now()
}

type RunProgress struct {
	Done   shared.HeightRange `json:"done"`
	D      time.Duration      `json:"duration"`
	T      time.Time          `json:"time"`
	Errors []error            `json:"errors"`
}
