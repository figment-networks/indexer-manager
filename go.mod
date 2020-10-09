module github.com/figment-networks/cosmos-indexer

go 1.14

require (
	github.com/cosmos/cosmos-sdk v0.39.1
	github.com/figment-networks/indexing-engine v0.1.11
	github.com/golang-migrate/migrate/v4 v4.13.0
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lib/pq v1.8.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/go-amino v0.16.0
	github.com/tendermint/tendermint v0.33.8
	github.com/terra-project/core v0.4.0
	go.uber.org/zap v1.16.0
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	google.golang.org/grpc v1.32.0
	google.golang.org/protobuf v1.25.0
)

replace github.com/CosmWasm/go-cosmwasm => github.com/terra-project/go-cosmwasm v0.10.1-terra
