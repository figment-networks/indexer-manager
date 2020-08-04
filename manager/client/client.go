package client

import (
	"context"
)

// HubbleContractor a format agnostic
type HubbleContractor interface {
	GetCurrentHeight(ctx context.Context)
	GetCurrentBlock(ctx context.Context)
	GetBlock(ctx context.Context, id string)
	GetBlocks(ctx context.Context)
	GetBlockTimes(ctx context.Context)
	GetBlockTimesInterval(ctx context.Context)
	GetTransaction(ctx context.Context)
	GetTransactions(ctx context.Context)
	GetAccounts(ctx context.Context)
	GetAccount(ctx context.Context)
}

type HubbleClient struct {
}

func NewHubbleClient() *HubbleClient {
	return &HubbleClient{}
}

func (hc *HubbleClient) GetCurrentHeight(ctx context.Context) {

}

func (hc *HubbleClient) GetCurrentBlock(ctx context.Context) {

}

func (hc *HubbleClient) GetBlock(ctx context.Context, id string) {

}

func (hc *HubbleClient) GetBlocks(ctx context.Context) {

}

func (hc *HubbleClient) GetBlockTimes(ctx context.Context) {

}

func (hc *HubbleClient) GetBlockTimesInterval(ctx context.Context) {

}

func (hc *HubbleClient) GetTransaction(ctx context.Context) {

}

func (hc *HubbleClient) GetTransactions(ctx context.Context) {

}

func (hc *HubbleClient) GetAccounts(ctx context.Context) {

}

func (hc *HubbleClient) GetAccount(ctx context.Context) {

}

func (hc *HubbleClient) RouteMessage() {

}
