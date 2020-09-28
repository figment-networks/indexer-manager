package cosmos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/figment-networks/cosmos-indexer/structs"
	shared "github.com/figment-networks/cosmos-indexer/structs"
	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
	"github.com/figment-networks/indexing-engine/metrics"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
	log "github.com/sirupsen/logrus"
)

// TxLogError Error message
type TxLogError struct {
	Codespace string  `json:"codespace"`
	Code      float64 `json:"code"`
	Message   string  `json:"message"`
}

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-\\s]+)([^0-9\\s]+)$")

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, r structs.HeightRange, blocks map[uint64]shared.Block, out chan cStruct.OutResp, page, perPage int, fin chan string) {
	defer c.logger.Sync()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		fin <- err.Error()
		return
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()
	s := strings.Builder{}

	s.WriteString(`"`)

	if r.EndHeight > 0 && r.EndHeight != r.StartHeight {
		s.WriteString("tx.height>=")
		s.WriteString(strconv.FormatUint(r.StartHeight, 10))
		s.WriteString(" AND tx.height<=")
		s.WriteString(strconv.FormatUint(r.EndHeight, 10))
	} else {
		s.WriteString("tx.height=")
		s.WriteString(strconv.FormatUint(r.StartHeight, 10))
	}
	s.WriteString(`"`)

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	// (lukanus): do not block initial calls
	if r.EndHeight != 0 && r.StartHeight != 0 {
		err = c.rateLimiter.Wait(ctx)
		if err != nil {
			fin <- err.Error()
			return
		}
	}

	now := time.Now()
	resp, err := c.httpClient.Do(req)

	log.Debug("[COSMOS-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
	if err != nil {
		fin <- err.Error()
		return
	}

	if resp.StatusCode > 399 { // ERROR
		serverError, _ := ioutil.ReadAll(resp.Body)

		c.logger.Error("[COSMOS-API] error getting response from server", zap.Int("code", resp.StatusCode), zap.Any("response", string(serverError)))
		err := fmt.Errorf("error getting response from server %d %s", resp.StatusCode, string(serverError))
		fin <- err.Error()
		return
	}

	rawRequestDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		c.logger.Error("[COSMOS-API] unable to decode result body", zap.Error(err))
		err := fmt.Errorf("unable to decode result body %w", err)
		fin <- err.Error()
		return
	}

	if result.Error.Message != "" {
		c.logger.Error("[COSMOS-API] Error getting search", zap.Any("result", result.Error.Message))
		err := fmt.Errorf("Error getting search: %s", result.Error.Message)
		fin <- err.Error()
		return
	}

	totalCount, err := strconv.ParseInt(result.Result.TotalCount, 10, 64)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery), zap.Any("request", r))
		fin <- err.Error()
		return
	}

	numberOfItemsTransactions.Observe(float64(totalCount))

	c.logger.Debug("[COSMOS-API] Converting requests ", zap.Int("number", len(result.Result.Txs)), zap.Int("blocks", len(blocks)))
	err = rawToTransaction(ctx, c, result.Result.Txs, blocks, out, c.logger, c.cdc)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting rawToTransaction", zap.Error(err))
		fin <- err.Error()
	}
	c.logger.Debug("[COSMOS-API] Converted all requests ")

	fin <- ""
	return
}

// transform raw data from cosmos into transaction format with augmentation from blocks
func rawToTransaction(ctx context.Context, c *Client, in []TxResponse, blocks map[uint64]shared.Block, out chan cStruct.OutResp, logger *zap.Logger, cdc *codec.Codec) error {
	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	for _, txRaw := range in {
		timer := metrics.NewTimer(transactionConversionDuration)

		tx := &auth.StdTx{}

		readr.Reset(txRaw.TxResult.Log)
		lf := []LogFormat{}
		txErr := TxLogError{}
		err := dec.Decode(&lf)
		if err != nil {
			// (lukanus): Try to fallback to known error format
			readr.Reset(txRaw.TxResult.Log)
			errin := dec.Decode(&txErr)
			if errin != nil {
				logger.Error("[COSMOS-API] Problem decoding raw transaction (json)", zap.Error(err), zap.String("content_log", txRaw.TxResult.Log), zap.Any("content", txRaw))
				continue
			}
		}
		txReader := strings.NewReader(txRaw.TxData)
		base64Dec := base64.NewDecoder(base64.StdEncoding, txReader)
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
		if err != nil {
			logger.Error("[COSMOS-API] Problem decoding raw transaction (cdc) ", zap.Error(err))
		}
		hInt, err := strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			logger.Error("[COSMOS-API] Problem parsing height", zap.Error(err))
		}

		outTX := cStruct.OutResp{Type: "Transaction"}
		block := blocks[hInt]

		trans := shared.Transaction{
			Hash:      txRaw.Hash,
			Memo:      tx.GetMemo(),
			Time:      block.Time,
			ChainID:   block.ChainID,
			BlockHash: block.Hash,
		}

		trans.Height, err = strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasWanted, err = strconv.ParseUint(txRaw.TxResult.GasWanted, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasUsed, err = strconv.ParseUint(txRaw.TxResult.GasUsed, 10, 64)
		if err != nil {
			outTX.Error = err
		}

		txReader.Seek(0, 0)
		trans.Raw = make([]byte, txReader.Len())
		txReader.Read(trans.Raw)

		presentIndexes := map[string]bool{}

		for index, msg := range tx.Msgs {
			tev := shared.TransactionEvent{
				ID: strconv.Itoa(index),
			}

			var ev shared.SubsetEvent
			var err error

			switch msg.Route() {
			case "bank":
				switch msg.Type() {
				case "multisend":
					ev, err = mapBankMultisendToSub(msg)
				case "send":
					ev, err = mapBankSendToSub(msg)
				default:
					c.logger.Error("[TERRA-API] Unknown bank message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "slashing":
				switch msg.Type() {
				case "unjail":
					ev, err = mapSlashingUnjailToSub(msg)
				default:
					c.logger.Error("[TERRA-API] Unknown slashing message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "gov":
				switch msg.Type() {
				case "deposit":
					ev, err = mapGovDepositToSub(msg)
				case "vote":
					ev, err = mapGovVoteToSub(msg)
				case "submit_proposal":
					ev, err = mapGovSubmitProposalToSub(msg)
				default:
					c.logger.Error("[TERRA-API] Unknown got message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "distribution":
				switch msg.Type() {
				case "withdraw_validator_commission":
					ev, err = mapDistributionWithdrawValidatorCommissionToSub(msg)
				case "set_withdraw_address":
					ev, err = mapDistributionSetWithdrawAddressToSub(msg)
				case "withdraw_delegator_reward":
					ev, err = mapDistributionWithdrawDelegatorRewardToSub(msg)
				case "fund_community_pool":
					ev, err = mapDistributionFundCommunityPoolToSub(msg)
				default:
					c.logger.Error("[TERRA-API] Unknown distribution message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			case "staking":
				switch msg.Type() {
				case "begin_unbonding":
					ev, err = mapStakingUndelegateToSub(msg)
				case "edit_validator":
					ev, err = mapStakingEditValidatorToSub(msg)
				case "create_validator":
					ev, err = mapStakingCreateValidatorToSub(msg)
				case "delegate":
					ev, err = mapStakingDelegateToSub(msg)
				case "begin_redelegate":
					ev, err = mapStakingBeginRedelegateToSub(msg)
				default:
					c.logger.Error("[TERRA-API] Unknown staking message Type ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
				}
			default:
				c.logger.Error("[TERRA-API] Unknown message Route ", zap.Error(err), zap.String("route", msg.Route()), zap.String("type", msg.Type()))
			}

			if len(ev.Type) > 0 {
				tev.Kind = msg.Type()
				tev.Sub = append(tev.Sub, ev)
			}

			if err != nil {
				c.logger.Error("[TERRA-API] Problem decoding transaction ", zap.Error(err), zap.String("type", msg.Type()), zap.String("route", msg.Route()))
			}

			presentIndexes[tev.ID] = true
			trans.Events = append(trans.Events, tev)
		}

		for _, logf := range lf {
			msgIndex := strconv.FormatFloat(logf.MsgIndex, 'f', -1, 64)
			_, ok := presentIndexes[msgIndex]
			if ok {
				continue
			}

			tev := shared.TransactionEvent{
				ID: msgIndex,
			}
			for _, ev := range logf.Events {
				sub := shared.SubsetEvent{
					Type: []string{ev.Type},
				}
				for atk, attr := range ev.Attributes {
					sub.Module = attr.Module
					sub.Action = attr.Action

					if len(attr.Sender) > 0 {
						for _, senderID := range attr.Sender {
							sub.Sender = append(sub.Sender, shared.EventTransfer{Account: shared.Account{ID: senderID}})
						}
					}
					if len(attr.Recipient) > 0 {
						for _, recipientID := range attr.Recipient {
							sub.Recipient = append(sub.Recipient, shared.EventTransfer{Account: shared.Account{ID: recipientID}})
						}
					}
					if attr.CompletionTime != "" {
						cTime, _ := time.Parse(time.RFC3339Nano, attr.CompletionTime)
						sub.Completion = &cTime
					}
					if len(attr.Validator) > 0 {
						if sub.Node == nil {
							sub.Node = make(map[string][]shared.Account)
						}
						for k, v := range attr.Validator {
							w, ok := sub.Node[k]
							if !ok {
								w = []shared.Account{}
							}

							for _, validatorID := range v {
								w = append(w, shared.Account{ID: validatorID})
							}
							sub.Node[k] = w
						}
					}

					for index, amount := range attr.Amount {
						sliced := getCurrency(amount)

						am := shared.TransactionAmount{
							Text: amount,
						}

						var (
							c       *big.Int
							exp     int32
							coinErr error
						)

						if len(sliced) == 3 {
							am.Currency = sliced[2]
							c, exp, coinErr = getCoin(sliced[1])
						} else {
							c, exp, coinErr = getCoin(amount)
						}

						if coinErr != nil {
							am.Numeric.Set(c)
							am.Exp = exp
						}
						sub.Amount[strconv.Itoa(index)] = am
					}
					ev.Attributes[atk] = nil
				}
				tev.Sub = append(tev.Sub, sub)

			}
			logf.Events = nil
			trans.Events = append(trans.Events, tev)

		}

		if txErr.Message != "" {
			tev := shared.TransactionEvent{
				Kind: "error",
				Sub: []shared.SubsetEvent{{
					Type:   []string{"error"},
					Module: txErr.Codespace,
					Error:  &shared.SubsetEventError{Message: txErr.Message},
				}},
			}
			trans.Events = append(trans.Events, tev)
		}

		outTX.Payload = trans
		out <- outTX
		timer.ObserveDuration()

		// GC Help
		lf = nil
	}

	return nil
}

func getCurrency(in string) []string {
	return curencyRegex.FindStringSubmatch(in)
}

func getCoin(s string) (number *big.Int, exp int32, err error) {

	s = strings.Replace(s, ",", ".", -1)
	strs := strings.Split(s, `.`)
	if len(strs) == 1 {
		i := &big.Int{}
		i.SetString(strs[0], 10)
		return i, 0, nil
	}
	if len(strs) == 2 {
		i := &big.Int{}
		i.SetString(strs[0]+strs[1], 10)
		return i, int32(len(strs[1])), nil
	}

	return number, 0, errors.New("Impossible to parse ")
}
