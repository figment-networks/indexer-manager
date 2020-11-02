# Change Log

# Change Log

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
