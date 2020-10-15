# COSMOS INDEXER/SEARCH

This repository contains an indexer for cosmos transactions. It consists of two parts: Manager and Worker.
Manager is process maintaining database and serving traffic. Worker acts as a stateless scraper.

## Structure

### Connectivity

Manager and Worker are linked by GRPC bidirectional streams. To connect with each other, worker needs to know manager's http interface. After initial request is done - GRPC connection is started in another direction - from client to manager. All the following communication between services happen after.  For every request, worker should sent a x number of responses marked with request's taskID. Response set should be terminated by one message with status being final.

### Activity

Workers are meant to be stateless, this is why the main responsibility of managers is to serve requests from outside. Managers are not creating intermediate states. Every operation done on data is atomic. This allows to run more than one instance of manager in the same time.

### Scheduling

To fulfill current goals, system needs to operate on actions that are triggered periodically.  Because system may work with multiple instances of manager, this creates an obvious problem of reoccurring activities. This is why scheduler was created as separated package. It allows to connect to multiple managers in the form of SDS-like service. For the simplest scenarios with only one manager - scheduler is embedded into manager binary, skipping connectivity part.


## Manager
Process that is responsible for maintaining current state of system. It is operating with the persistence layer. It is also serving traffic from/to external users.

### Client
Manager's business logic. It should not contain any transport related operations such as data conversions of parameter checks. All the input and output structures should be as  carrier-agnostic as possible.

### Connectivity
Manager's Package for maintaining connections with workers. Currently allows to start the abstract connection with workers, match request with the response and direct that to proper client handles. Also implements simple round robin strategy for sending every next request to another currently connected worker.

### Store
Persistent layer with only database operations. This package should not contain any business logic checks. Currently supports postgres database.

### Transport
- GRPC transport for connectivity package. Bidirectional stream implementation for worker connectivity.
- HTTP transport containing set of user-facing endpoints with non-business logic data validation.

## Worker

Stateless worker is responsible for connecting with the chain, getting information, converting it to a common format and sending it back to manager.
Worker can be connected with multiple managers but should always answer only to the one that sent request.

### Transport
Currently implemented transport allow worker to run using grpc.

### Connectivity
Basic functions for initial simple service discovery. Right now, http queries to every manager triggered periodically.

## API
Implementation of bare requests for network.

### Client
Worker's business logic wiring of messages to client's functions.


## Installation
This system can be put together in many different ways.
This readme will describe only the simplest one worker, one manager with embedded scheduler approach.

### Docker-Compose
This repo also comes with partially preconfigured docker-compose setup.
To run using docker-compose you need to
- Create scheduler configuration as stated below in ./scheduler/scheduler.json (or empty structure if you don't need scraping)
```json
[
    {
        "network": "cosmos",
        "chainID": "cosmoshub-3",
        "version": "0.0.1",
        "duration": "30s",
        "kind": "lastdata"
    }
]
```
- Start docker-compose (If you're running that for the first time you may need to run `docker swarm init` )
```bash
    docker-compose build
    docker-compose up
```

First manager would (the main api) would be available under `http://0.0.0.0:8085`

Docker-compose version comes also with preconfigured prometheus and grafana, serving content on `:3000` and `:9090`

Tip: To purge database simply run `docker volume rm indexer-manager_postgresdatabase` after fully stopping the run

### Compile
To compile sources you need to have go 1.14.1+ installed. For Worker and Manager it's respectively

```bash
    make build-cosmos
    make build-manager-migration
    make build-manager-w-scheduler
```

### Running
The mandatory env variables (or json file) for both manager with embedded scheduler and migration the parameters are:
```bash
    ADDRESS=0.0.0.0:8085
    DATABASE_URL=postgres://cosmos:cosmos@cosmosdatabase/cosmos?sslmode=disable
    SCHEDULER_INITIAL_CONFIG_PATH=./schedules/
    ENABLE_SCHEDULER=true
```
Where `ADDRESS` is the ip:port string with http interface.
It is possible to set the same parameters in json file under `--config config.json`


After setting up the database you need to run migration script it takes database from DATABASE_URL env var. (migrations are inside `cmd/manager-migration/migrations` directory)

```bash
    manager_migration_bin  --path "./path/to/migrations"
```

The embedded scheduler still needs it's configuration to scrape cosmos, cosmoshub-3 for the last data, every 30s  the `scheduler.json` file needs to be would be following:

```json
[
    {
        "network": "cosmos",
        "chainID": "cosmoshub-3",
        "version": "0.0.1",
        "duration": "30s",
        "kind": "lastdata"
    }
]
```
Where `duration` is a `time.Duration` parsed string, so valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". (https://golang.org/pkg/time/#ParseDuration)

After setting all of it just run the binary.

Worker also need some basic config:

```bash
    MANAGERS=0.0.0.0:8085
    TENDERMINT_RPC_ADDR=https://cosmoshub-3.address
    DATAHUB_KEY=1QAZXSW23EDCvfr45TGB
    CHAIN_ID=cosmoshub-3
```
Where
    - `TENDERMINT_RPC_ADDR` is a http address to node's RPC endpoint
    - `MANAGERS` a comma-separated list of manager ip:port addresses that worker will connect to. In this case only one

After running both binaries worker should successfully register itself to the manager.

### Initial Chain Synchronization
Scheduler allows to apply many different events to be emitted periodically over time. One of such is `lastdata` which is used for getting latest records from chain.
Running `lastdata` would effect in taking up to 1000 last heights that system haven't scraped yet.
Every time scheduler runs, it checks fot last scrape information and starts from there. If there isn't any scraped data yet, process is starting from last block minus 1000 heights. If current height is smaller than 1000 - it starts from the first block.

To synchronize with missing height range you need to manually start the process by calling the `/check_missing` and/or `/get_missing` endpoints.

To check missing transactions simply call manager as:

```GET http://0.0.0.0:8085/check_missing?start_height=1&end_height=3500000&network=cosmos&chain_id=cosmoshub-3```

Where `start_height`/`end_height` is range *existing* in chain.

The `get_missing` endpoint is prepared to run big queries asynchronously. It is running as goroutines and it's not persisting any state. In case of process crash, just rerun operation. It should start with consistency check and begin scraping from the highest height it didn't have.

To start scraping, make following request within *existing* range.

```GET http://0.0.0.0:8085/get_missing?force=true&async=true&start_height=1&end_height=3500000&network=cosmos&chain_id=cosmoshub-3```

Where
    - `force` indicate the start/restart of scraping process. The next call without this parameter will return current state of scraping
    - `async` runs scraping process as goroutine. otherwise the process will die on any network timeout.

### Transaction Search
Transaction Search is available by making a post request on `/transaction_search`.
The detailed description of parameters and response is available in [swagger.json](./swagger.json).

Current implementation allows to query entire range, however some complex queries take time.
You can however, speed up every query by setting `after_time`, `before_time` or `after_height`, `before_height` parameters to narrow down known time/height range.
All times are RFC3339.

List of cosmos defined types (listed by modules):
- bank:
    `multisend` , `send`
- crisis:
    `verify_invariant`
- distribution:
    `withdraw_validator_commission` , `set_withdraw_address` , `withdraw_delegator_reward` , `fund_community_pool`
- evidence:
    `submit_evidence`
- gov:
    `deposit` , `vote` , `submit_proposal`
- slashing:
    `unjail`
- staking:
    `begin_unbonding` , `edit_validator` , `create_validator` , `delegate` , `begin_redelegate`
- internal:
    `error`