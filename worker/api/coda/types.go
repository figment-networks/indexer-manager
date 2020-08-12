// AUTOGENERATED
// TODO: WIRE UP AUTOGENERATOR

package coda

import (
	"fmt"
	"io"
	"strconv"
)

// An account record according to the daemon
type Account struct {
	// The public identity of the account
	PublicKey string `json:"publicKey"`
	// The amount of coda owned by the account
	Balance *AnnotatedBalance `json:"balance"`
	// A natural number that increases with each transaction (stringified uint32)
	Nonce *string `json:"nonce"`
	// Like the `nonce` field, except it includes the scheduled transactions (transactions not yet included in a block) (stringified uint32)
	InferredNonce *string `json:"inferredNonce"`
	// The account that you delegated on the staking ledger of the current block's epoch
	EpochDelegateAccount *Account `json:"epochDelegateAccount"`
	// Top hash of the receipt chain merkle-list
	ReceiptChainHash *string `json:"receiptChainHash"`
	// The public key to which you are delegating - if you are not delegating to anybody, this would return your public key
	Delegate *string `json:"delegate"`
	// The account to which you are delegating - if you are not delegating to anybody, this would return your public key
	DelegateAccount *Account `json:"delegateAccount"`
	// The list of accounts which are delegating to you (note that the info is recorded in the last epoch so it might not be up to date with the current account status)
	Delegators []*Account `json:"delegators"`
	// The list of accounts which are delegating to you in the last epoch (note that the info is recorded in the one before last epoch epoch so it might not be up to date with the current account status)
	LastEpochDelegators []*Account `json:"lastEpochDelegators"`
	// The previous epoch lock hash of the chain which you are voting for
	VotingFor *string `json:"votingFor"`
	// True if you are actively staking with this account on the current daemon - this may not yet have been updated if the staking key was changed recently
	StakingActive bool `json:"stakingActive"`
	// Path of the private key file for this account
	PrivateKeyPath string `json:"privateKeyPath"`
	// True if locked, false if unlocked, null if the account isn't tracked by the queried daemon
	Locked *bool `json:"locked"`
}

type AddAccountInput struct {
	// Password used to encrypt the new account
	Password string `json:"password"`
}

type AddAccountPayload struct {
	// Public key of the created account
	PublicKey string `json:"publicKey"`
	// Details of created account
	Account *Account `json:"account"`
}

type AddPaymentReceiptInput struct {
	// Time that a payment gets added to another clients transaction database (stringified Unix time - number of milliseconds since January 1, 1970)
	AddedTime string `json:"added_time"`
	// Serialized payment (base58-encoded janestreet/bin_prot serialization)
	Payment string `json:"payment"`
}

type AddPaymentReceiptPayload struct {
	Payment *UserCommand `json:"payment"`
}

type AddrsAndPorts struct {
	ExternalIP string `json:"externalIp"`
	BindIP     string `json:"bindIp"`
	Peer       *Peer  `json:"peer"`
	Libp2pPort int    `json:"libp2pPort"`
	ClientPort int    `json:"clientPort"`
}

// A total balance annotated with the amount that is currently unknown with the invariant: unknown <= total
type AnnotatedBalance struct {
	// The amount of coda owned by the account
	Total string `json:"total"`
	// The amount of coda owned by the account whose origin is currently unknown
	Unknown string `json:"unknown"`
	// Block height at which balance was measured
	BlockHeight string `json:"blockHeight"`
}

type Block struct {
	// Public key of account that produced this block
	Creator string `json:"creator"`
	// Account that produced this block
	CreatorAccount *Account `json:"creatorAccount"`
	// Base58Check-encoded hash of the state after this block
	StateHash string `json:"stateHash"`
	// Experimental: Bigint field-element representation of stateHash
	StateHashField string         `json:"stateHashField"`
	ProtocolState  *ProtocolState `json:"protocolState"`
	// Snark proof of blockchain state
	ProtocolStateProof *ProtocolStateProof `json:"protocolStateProof"`
	Transactions       *Transactions       `json:"transactions"`
	SnarkJobs          []*CompletedWork    `json:"snarkJobs"`
}

// Connection as described by the Relay connections spec
type BlockConnection struct {
	Edges      []*BlockEdge `json:"edges"`
	Nodes      []*Block     `json:"nodes"`
	TotalCount int          `json:"totalCount"`
	PageInfo   *PageInfo    `json:"pageInfo"`
}

// Connection Edge as described by the Relay connections spec
type BlockEdge struct {
	// Opaque pagination cursor for a block (base58-encoded janestreet/bin_prot serialization)
	Cursor string `json:"cursor"`
	Node   *Block `json:"node"`
}

type BlockFilterInput struct {
	// A public key of a user who has their
	//         transaction in the block, or produced the block
	RelatedTo string `json:"relatedTo"`
}

type BlockProducerTimings struct {
	Times []*ConsensusTime `json:"times"`
}

type BlockchainState struct {
	// date (stringified Unix time - number of milliseconds since January 1, 1970)
	Date string `json:"date"`
	// Base58Check-encoded hash of the snarked ledger
	SnarkedLedgerHash string `json:"snarkedLedgerHash"`
	// Base58Check-encoded hash of the staged ledger
	StagedLedgerHash string `json:"stagedLedgerHash"`
}

// Completed snark works
type CompletedWork struct {
	// Public key of the prover
	Prover string `json:"prover"`
	// Amount the prover is paid for the snark work
	Fee string `json:"fee"`
	// Unique identifier for the snark work purchased
	WorkIds []int `json:"workIds"`
}

type ConsensusConfiguration struct {
	Delta                  int    `json:"delta"`
	K                      int    `json:"k"`
	C                      int    `json:"c"`
	CTimesK                int    `json:"cTimesK"`
	SlotsPerEpoch          int    `json:"slotsPerEpoch"`
	SlotDuration           int    `json:"slotDuration"`
	EpochDuration          int    `json:"epochDuration"`
	GenesisStateTimestamp  string `json:"genesisStateTimestamp"`
	AcceptableNetworkDelay int    `json:"acceptableNetworkDelay"`
}

type ConsensusState struct {
	// Length of the blockchain at this block
	BlockchainLength string `json:"blockchainLength"`
	// Height of the blockchain at this block
	BlockHeight      string `json:"blockHeight"`
	EpochCount       string `json:"epochCount"`
	MinWindowDensity string `json:"minWindowDensity"`
	LastVrfOutput    string `json:"lastVrfOutput"`
	// Total currency in circulation at this block
	TotalCurrency                     string            `json:"totalCurrency"`
	StakingEpochData                  *StakingEpochData `json:"stakingEpochData"`
	NextEpochData                     *NextEpochData    `json:"nextEpochData"`
	HasAncestorInSameCheckpointWindow bool              `json:"hasAncestorInSameCheckpointWindow"`
	// Slot in which this block was created
	Slot string `json:"slot"`
	// Epoch in which this block was created
	Epoch string `json:"epoch"`
}

type ConsensusTime struct {
	Epoch      string `json:"epoch"`
	Slot       string `json:"slot"`
	GlobalSlot string `json:"globalSlot"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type CreateHDAccountInput struct {
	// Index of the account in hardware wallet
	Index string `json:"index"`
}

type DaemonStatus struct {
	NumAccounts                *int                    `json:"numAccounts"`
	BlockchainLength           *int                    `json:"blockchainLength"`
	HighestBlockLengthReceived int                     `json:"highestBlockLengthReceived"`
	UptimeSecs                 int                     `json:"uptimeSecs"`
	LedgerMerkleRoot           *string                 `json:"ledgerMerkleRoot"`
	StateHash                  *string                 `json:"stateHash"`
	CommitID                   string                  `json:"commitId"`
	ConfDir                    string                  `json:"confDir"`
	Peers                      []string                `json:"peers"`
	UserCommandsSent           int                     `json:"userCommandsSent"`
	SnarkWorker                *string                 `json:"snarkWorker"`
	SnarkWorkFee               int                     `json:"snarkWorkFee"`
	SyncStatus                 SyncStatus              `json:"syncStatus"`
	BlockProductionKeys        []string                `json:"blockProductionKeys"`
	Histograms                 *Histograms             `json:"histograms"`
	ConsensusTimeBestTip       *ConsensusTime          `json:"consensusTimeBestTip"`
	NextBlockProduction        *BlockProducerTimings   `json:"nextBlockProduction"`
	ConsensusTimeNow           *ConsensusTime          `json:"consensusTimeNow"`
	ConsensusMechanism         string                  `json:"consensusMechanism"`
	ConsensusConfiguration     *ConsensusConfiguration `json:"consensusConfiguration"`
	AddrsAndPorts              *AddrsAndPorts          `json:"addrsAndPorts"`
}

type DeleteAccountInput struct {
	// Public key of account to be deleted
	PublicKey string `json:"publicKey"`
}

type DeleteAccountPayload struct {
	// Public key of the deleted account
	PublicKey string `json:"publicKey"`
}

type FeeTransfer struct {
	// Public key of fee transfer recipient
	Recipient string `json:"recipient"`
	// Amount that the recipient is paid in this fee transfer
	Fee string `json:"fee"`
}

type Histogram struct {
	Values    []int       `json:"values"`
	Intervals []*Interval `json:"intervals"`
	Underflow int         `json:"underflow"`
	Overflow  int         `json:"overflow"`
}

type Histograms struct {
	RPCTimings                      *RPCTimings `json:"rpcTimings"`
	ExternalTransitionLatency       *Histogram  `json:"externalTransitionLatency"`
	AcceptedTransitionLocalLatency  *Histogram  `json:"acceptedTransitionLocalLatency"`
	AcceptedTransitionRemoteLatency *Histogram  `json:"acceptedTransitionRemoteLatency"`
	SnarkWorkerTransitionTime       *Histogram  `json:"snarkWorkerTransitionTime"`
	SnarkWorkerMergeTime            *Histogram  `json:"snarkWorkerMergeTime"`
}

type Interval struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type LockInput struct {
	// Public key specifying which account to lock
	PublicKey string `json:"publicKey"`
}

type LockPayload struct {
	// Public key of the locked account
	PublicKey string `json:"publicKey"`
	// Details of locked account
	Account *Account `json:"account"`
}

type NextEpochData struct {
	Ledger          *EpochLedger `json:"ledger"`
	Seed            string       `json:"seed"`
	StartCheckpoint string       `json:"startCheckpoint"`
	LockCheckpoint  string       `json:"lockCheckpoint"`
	EpochLength     string       `json:"epochLength"`
}

// PageInfo object as described by the Relay connections spec
type PageInfo struct {
	HasPreviousPage bool    `json:"hasPreviousPage"`
	HasNextPage     bool    `json:"hasNextPage"`
	FirstCursor     *string `json:"firstCursor"`
	LastCursor      *string `json:"lastCursor"`
}

type Peer struct {
	Host       string `json:"host"`
	Libp2pPort int    `json:"libp2pPort"`
	PeerID     string `json:"peerId"`
}

// Snark work bundles that are not available in the pool yet
type PendingSnarkWork struct {
	// Work bundle with one or two snark work
	WorkBundle []*WorkDescription `json:"workBundle"`
}

type ProtocolState struct {
	// Base58Check-encoded hash of the previous state
	PreviousStateHash string `json:"previousStateHash"`
	// State which is agnostic of a particular consensus algorithm
	BlockchainState *BlockchainState `json:"blockchainState"`
	// State specific to the Codaboros Proof of Stake consensus algorithm
	ConsensusState *ConsensusState `json:"consensusState"`
}

type ReloadAccountsPayload struct {
	// True when the reload was successful
	Success bool `json:"success"`
}

type RPCPair struct {
	Dispatch *Histogram `json:"dispatch"`
	Impl     *Histogram `json:"impl"`
}

type RPCTimings struct {
	GetStagedLedgerAux      *RPCPair `json:"getStagedLedgerAux"`
	AnswerSyncLedgerQuery   *RPCPair `json:"answerSyncLedgerQuery"`
	GetAncestry             *RPCPair `json:"getAncestry"`
	GetTransitionChainProof *RPCPair `json:"getTransitionChainProof"`
	GetTransitionChain      *RPCPair `json:"getTransitionChain"`
}

type SendDelegationInput struct {
	// Should only be set when cancelling transactions, otherwise a nonce is determined automatically
	Nonce *string `json:"nonce"`
	// Short arbitrary message provided by the sender
	Memo *string `json:"memo"`
	// The global slot number after which this transaction cannot be applied
	ValidUntil *string `json:"validUntil"`
	// Fee amount in order to send a stake delegation
	Fee string `json:"fee"`
	// Public key of the account being delegated to
	To string `json:"to"`
	// Public key of sender of a stake delegation
	From string `json:"from"`
}

type SendDelegationPayload struct {
	// Delegation change that was sent
	Delegation *UserCommand `json:"delegation"`
}

type SendPaymentInput struct {
	// Should only be set when cancelling transactions, otherwise a nonce is determined automatically
	Nonce *string `json:"nonce"`
	// Short arbitrary message provided by the sender
	Memo *string `json:"memo"`
	// The global slot number after which this transaction cannot be applied
	ValidUntil *string `json:"validUntil"`
	// Fee amount in order to send payment
	Fee string `json:"fee"`
	// Amount of coda to send to to receiver
	Amount string `json:"amount"`
	// Public key of recipient of payment
	To string `json:"to"`
	// Public key of sender of payment
	From string `json:"from"`
}

type SendPaymentPayload struct {
	// Payment that was sent
	Payment *UserCommand `json:"payment"`
}

type SetSnarkWorkFee struct {
	// Fee to get rewarded for producing snark work
	Fee string `json:"fee"`
}

type SetSnarkWorkFeePayload struct {
	// Returns the last fee set to do snark work
	LastFee string `json:"lastFee"`
}

type SetSnarkWorkerInput struct {
	// Public key you wish to start snark-working on; null to stop doing any snark work
	PublicKey *string `json:"publicKey"`
}

type SetSnarkWorkerPayload struct {
	// Returns the last public key that was designated for snark work
	LastSnarkWorker *string `json:"lastSnarkWorker"`
}

type SetStakingInput struct {
	// Public keys of accounts you wish to stake with - these must be accounts that are in trackedAccounts
	PublicKeys []string `json:"publicKeys"`
}

type SetStakingPayload struct {
	// Returns the public keys that were staking funds previously
	LastStaking []string `json:"lastStaking"`
	// List of public keys that could not be used to stake because they were locked
	LockedPublicKeys []string `json:"lockedPublicKeys"`
	// Returns the public keys that are now staking their funds
	CurrentStakingKeys []string `json:"currentStakingKeys"`
}

// A cryptographic signature
type SignatureInput struct {
	// Scalar component of signature
	Scalar string `json:"scalar"`
	// Field component of signature
	Field string `json:"field"`
}

// Signed fee
type SignedFee struct {
	// +/-
	Sign Sign `json:"sign"`
	// Fee
	FeeMagnitude string `json:"feeMagnitude"`
}

type SnarkWorker struct {
	// Public key of current snark worker
	Key string `json:"key"`
	// Account of the current snark worker
	Account *Account `json:"account"`
	// Fee that snark worker is charging to generate a snark proof
	Fee string `json:"fee"`
}

type StakingEpochData struct {
	Ledger          *EpochLedger `json:"ledger"`
	Seed            string       `json:"seed"`
	StartCheckpoint string       `json:"startCheckpoint"`
	LockCheckpoint  string       `json:"lockCheckpoint"`
	EpochLength     string       `json:"epochLength"`
}

// Different types of transactions in a block
type Transactions struct {
	// List of user commands (payments and stake delegations) included in this block
	UserCommands []*UserCommand `json:"userCommands"`
	// List of fee transfers included in this block
	FeeTransfer []*FeeTransfer `json:"feeTransfer"`
	// Amount of coda granted to the producer of this block
	Coinbase string `json:"coinbase"`
	// Receiver of the block reward
	CoinbaseReceiver *Account `json:"coinbaseReceiverAccount"`
}

type TrustStatusPayload struct {
	// IP address
	IPAddr string `json:"ip_addr"`
	// Trust score
	Trust float64 `json:"trust"`
	// Banned status
	BannedStatus *string `json:"banned_status"`
}

type UnlockInput struct {
	// Public key specifying which account to unlock
	PublicKey string `json:"publicKey"`
	// Password for the account to be unlocked
	Password string `json:"password"`
}

type UnlockPayload struct {
	// Public key of the unlocked account
	PublicKey string `json:"publicKey"`
	// Details of unlocked account
	Account *Account `json:"account"`
}

type UserCommand struct {
	ID string `json:"id"`
	// If true, this represents a delegation of stake, otherwise it is a payment
	IsDelegation bool `json:"isDelegation"`
	// Nonce of the transaction
	Nonce int `json:"nonce"`
	// Public key of the sender
	From string `json:"from"`
	// Account of the sender
	FromAccount *Account `json:"fromAccount"`
	// Public key of the receiver
	To string `json:"to"`
	// Account of the receiver
	ToAccount *Account `json:"toAccount"`
	// Amount that sender is sending to receiver - this is 0 for delegations
	Amount string `json:"amount"`
	// Fee that sender is willing to pay for making the transaction
	Fee string `json:"fee"`
	// Short arbitrary message provided by the sender
	Memo string `json:"memo"`
}

// Transition from a source ledger to a target ledger with some fee excess and increase in supply
type WorkDescription struct {
	// Base58Check-encoded hash of the source ledger
	SourceLedgerHash string `json:"sourceLedgerHash"`
	// Base58Check-encoded hash of the target ledger
	TargetLedgerHash string `json:"targetLedgerHash"`
	// Total transaction fee that is not accounted for in the transition from source ledger to target ledger
	FeeExcess *SignedFee `json:"feeExcess"`
	// Increase in total coinbase reward
	SupplyIncrease string `json:"supplyIncrease"`
	// Unique identifier for a snark work
	WorkID int `json:"workId"`
}

type EpochLedger struct {
	Hash          string `json:"hash"`
	TotalCurrency string `json:"totalCurrency"`
}

type ProtocolStateProof struct {
	A          []string   `json:"a"`
	B          [][]string `json:"b"`
	C          []string   `json:"c"`
	DeltaPrime [][]string `json:"delta_prime"`
	Z          []string   `json:"z"`
}

// Status for whenever the blockchain is reorganized
type ChainReorganizationStatus string

const (
	ChainReorganizationStatusChanged ChainReorganizationStatus = "CHANGED"
)

var AllChainReorganizationStatus = []ChainReorganizationStatus{
	ChainReorganizationStatusChanged,
}

func (e ChainReorganizationStatus) IsValid() bool {
	switch e {
	case ChainReorganizationStatusChanged:
		return true
	}
	return false
}

func (e ChainReorganizationStatus) String() string {
	return string(e)
}

func (e *ChainReorganizationStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ChainReorganizationStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ChainReorganizationStatus", str)
	}
	return nil
}

func (e ChainReorganizationStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Sync status of daemon
type SyncStatus string

const (
	SyncStatusConnecting SyncStatus = "CONNECTING"
	SyncStatusListening  SyncStatus = "LISTENING"
	SyncStatusOffline    SyncStatus = "OFFLINE"
	SyncStatusBootstrap  SyncStatus = "BOOTSTRAP"
	SyncStatusSynced     SyncStatus = "SYNCED"
	SyncStatusCatchup    SyncStatus = "CATCHUP"
)

var AllSyncStatus = []SyncStatus{
	SyncStatusConnecting,
	SyncStatusListening,
	SyncStatusOffline,
	SyncStatusBootstrap,
	SyncStatusSynced,
	SyncStatusCatchup,
}

func (e SyncStatus) IsValid() bool {
	switch e {
	case SyncStatusConnecting, SyncStatusListening, SyncStatusOffline, SyncStatusBootstrap, SyncStatusSynced, SyncStatusCatchup:
		return true
	}
	return false
}

func (e SyncStatus) String() string {
	return string(e)
}

func (e *SyncStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = SyncStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid SyncStatus", str)
	}
	return nil
}

func (e SyncStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// Status of a transaction
type TransactionStatus string

const (
	// A transaction that is on the longest chain
	TransactionStatusIncluded TransactionStatus = "INCLUDED"
	// A transaction either in the transition frontier or in transaction pool but is not on the longest chain
	TransactionStatusPending TransactionStatus = "PENDING"
	// The transaction has either been snarked, reached finality through consensus or has been dropped
	TransactionStatusUnknown TransactionStatus = "UNKNOWN"
)

var AllTransactionStatus = []TransactionStatus{
	TransactionStatusIncluded,
	TransactionStatusPending,
	TransactionStatusUnknown,
}

func (e TransactionStatus) IsValid() bool {
	switch e {
	case TransactionStatusIncluded, TransactionStatusPending, TransactionStatusUnknown:
		return true
	}
	return false
}

func (e TransactionStatus) String() string {
	return string(e)
}

func (e *TransactionStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = TransactionStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid TransactionStatus", str)
	}
	return nil
}

func (e TransactionStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type Sign string

const (
	SignPlus  Sign = "PLUS"
	SignMinus Sign = "MINUS"
)

var AllSign = []Sign{
	SignPlus,
	SignMinus,
}

func (e Sign) IsValid() bool {
	switch e {
	case SignPlus, SignMinus:
		return true
	}
	return false
}

func (e Sign) String() string {
	return string(e)
}

func (e *Sign) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Sign(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid sign", str)
	}
	return nil
}

func (e Sign) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
