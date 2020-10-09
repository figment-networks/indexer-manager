package client

import (
	"context"
	"errors"
	"sync"
	"time"

	shared "github.com/figment-networks/indexer-manager/structs"
)

// Progresser is interface that allows to report progress
type Progresser interface {
	Report(done shared.HeightRange, duration time.Duration, err []error, finished bool)
}

// Runner is a runner for async long running process.
//
// Sometimes there is a need to run process asynchronously.
// Runner serves as a simple manager for NetworkVersion pairs
type Runner struct {
	runs map[NetworkVersion][]*Run
	in   chan RunReq
	out  chan chan ReqOut
}

// NewRunner is Runner constructor
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

// RunReq async task structure
type RunReq struct {
	Kind        string //Type
	NV          NetworkVersion
	HeightRange shared.HeightRange

	Force bool
	Resp  chan RunResp
}

// StartProcess starts task waiting until task will be started then return it's reference.
func (r *Runner) StartProcess(nv NetworkVersion, heightRange shared.HeightRange, force bool) (bool, *Run, error) {
	resp := make(chan RunResp, 1)
	defer close(resp)
	r.in <- RunReq{Kind: "start", NV: nv, HeightRange: heightRange, Force: force, Resp: resp}
	response := <-resp
	return response.IsNew, response.Run, response.Err
}

// StopProcess stops task
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
				if ok && currentlyRunning != nil && !rReq.Force {
					rReq.Resp <- RunResp{IsNew: false, Run: currentlyRunning, Err: nil}
					continue
				}

				if !ok || rReq.Force {
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
				rReq.Resp <- RunResp{IsNew: false, Run: nil, Err: errors.New("error occurred")}
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
	if ok {
		for _, run := range current {
			// totally inclusive
			if (req.HeightRange.StartHeight >= run.HeightRange.StartHeight && req.HeightRange.StartHeight <= run.HeightRange.EndHeight) &&
				(req.HeightRange.EndHeight <= run.HeightRange.EndHeight && req.HeightRange.EndHeight >= run.HeightRange.StartHeight) {
				return run, true
			}
			// TODO(lukanus): support other ranges (split), this is just optimization
		}
	}

	return nil, false
}

// Run is process handler
type Run struct {
	sync.Mutex `json:"-"`

	NV          NetworkVersion     `json:"nv"`
	HeightRange shared.HeightRange `json:"height_range"`

	Ctx    context.Context    `json:"-"`
	Cancel context.CancelFunc `json:"-"`

	Progress []RunProgress `json:"run_progress"`

	Success bool `json:"success"`

	Finished         bool      `json:"finished"`
	StartedTime      time.Time `json:"started_time"`
	FinishTime       time.Time `json:"finished_time"`
	LastProgressTime time.Time `json:"last_progress_time"`
}

// NewRun is Run constructor
func NewRun(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange) *Run {
	return &Run{
		Ctx:         ctx,
		NV:          nv,
		HeightRange: heightRange,
		Progress:    []RunProgress{},
	}
}

// Report reports progress of certain process
func (r *Run) Report(done shared.HeightRange, duration time.Duration, err []error, finished bool) {
	r.Lock()
	defer r.Unlock()

	now := time.Now()
	if r.StartedTime.IsZero() {
		r.StartedTime = now
	}
	r.LastProgressTime = now

	if finished {
		r.Success = true
		for _, p := range r.Progress {
			if len(p.Errors) > 0 {
				r.Success = false
			}
		}

		r.Finished = true
		r.FinishTime = now
	}

	if !finished || err != nil {
		r.Progress = append(r.Progress, RunProgress{Done: done, D: duration, T: time.Now(), Errors: err})
	}
}

// Stop stops.
func (r *Run) Stop() {
	r.Lock()
	defer r.Unlock()

	r.Cancel()
	r.Finished = true
	r.FinishTime = time.Now()
}

// RunProgress info
type RunProgress struct {
	Done   shared.HeightRange `json:"done"`
	D      time.Duration      `json:"duration"`
	T      time.Time          `json:"time"`
	Errors []error            `json:"errors"`
}
