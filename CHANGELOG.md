# Change Log

## [0.10.0] - 2021-03-03
### Added
- Implement `/account/balance` endpoint which returns daily balances of an account for a given time period.
### Changed
### Fixed

## [0.0.9] - 2021-01-28
### Added
- Implement `/rewards` endpoint which calculates daily rewards earned by an account for a given time period.
### Changed
### Fixed
## [0.0.8] - 2020-12-28
### Added
- 'transfers' field to 'sub' in transaction events. This field contains reward information.
### Changed
### Fixed

## [0.0.7] - 2020-12-03
### Added
- Ability to stop running sync process
### Changed
### Fixed

## [0.0.6] - 2020-12-01
### Added
- 'has_errors' field for transaction search return - contains a boolean describing error occurrence
- 'raw_log' field for transaction search return - contains a log information from transaction
- 'raw_log' column in database
- '/loglevel' endpoint for dynamically changing log level
### Changed
- '/get_missing' now has to two new parameters. 'simplified=true' returns the simplified version of progress log. 'overwrite_all=true' parameter allows to skip checking for data, and just download entire range regardless of what's inside the database
### Fixed

## [0.0.5] - 2020-11-17
### Added
- Indices for senders and receiver
- Populator util for populating changes in database (this version only for fee)
### Changed
- Fee is now correct type in database (jsonb) instead of decimal
- Senders/Receivers search now uses full text and then filters.
### Fixed
- Corrected wrong assignation of senders
- Fee assignations / return type / decode
- Hash search is fixed


## [0.0.4] - 2020-11-06

### Added
- Better handling of panics for rollbar
### Changed
### Fixed
- Problem (panic) with /check_missing in certain conditions


## [0.0.3] - 2020-11-02

### Added
### Changed
- Memo search is now taking place using full text search
- New database column was added for transactions
- Accounts are also conditionally added to full text as the support for array approach

### Fixed
- Decoder issue after error in the beginning of the transaction list.

## [0.0.2] - 2020-10-28

Here we would have the update steps for 1.2.4 for people to follow.

### Added
- Support for multiple chain ids queries to workers

### Changed
### Fixed
- Fixed offset parameters in search queries
- Removed client's StreamAccess deadlock by adding missed `sa.mapLock.Unlock()`

## [0.0.1] - 2020-10-20

Initial release

### Added
### Changed
### Fixed
