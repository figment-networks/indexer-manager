package embedded

import (
	"context"
	"fmt"

	"github.com/figment-networks/indexer-manager/scheduler/structures"
	"github.com/figment-networks/indexer-manager/structs"
)

type SchedulerContractor interface {
	ScrapeLatest(ctx context.Context, ldr structs.LatestDataRequest) (ldResp structs.LatestDataResponse, er error)
}

type LastDataInternalTransport struct {
	sc SchedulerContractor
}

func NewLastDataInternalTransport(sc SchedulerContractor) *LastDataInternalTransport {
	return &LastDataInternalTransport{
		sc: sc,
	}
}

func (ld *LastDataInternalTransport) GetLastData(ctx context.Context, ldReq structs.LatestDataRequest) (ldr structs.LatestDataResponse, err error) {
	ldr, err = ld.sc.ScrapeLatest(ctx, ldReq)
	if err != nil {
		return ldr, &structures.RunError{Contents: fmt.Errorf("error getting response from ScrapeLatest:  %w", err)}
	}

	return ldr, err
}
