package txpool

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/dogechain-lab/dogechain/chain"
	"github.com/dogechain-lab/dogechain/crypto"
	"github.com/dogechain-lab/dogechain/helper/tests"
	"github.com/dogechain-lab/dogechain/txpool/proto"
	"github.com/dogechain-lab/dogechain/types"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	defaultPriceLimit uint64 = 1
	defaultMaxSlots   uint64 = 4096
	validGasLimit     uint64 = 4712350
)

var (
	forks = &chain.Forks{
		Homestead: chain.NewFork(0),
		Istanbul:  chain.NewFork(0),
	}

	nilMetrics = NilMetrics()
)

// addresses used in tests
var (
	addr1 = types.Address{0x1}
	addr2 = types.Address{0x2}
	addr3 = types.Address{0x3}
	addr4 = types.Address{0x4}
	addr5 = types.Address{0x5}
)

// returns a new valid tx of slots size with the given nonce
func newTx(addr types.Address, nonce, slots uint64) *types.Transaction {
	return newPriceTx(
		addr,
		big.NewInt(0).SetUint64(defaultPriceLimit),
		nonce,
		slots,
	)
}

// returns a new valid tx of slots size with the given nonce and gas price
func newPriceTx(addr types.Address, price *big.Int, nonce, slots uint64) *types.Transaction {
	// base field should take 1 slot at least
	size := txSlotSize * (slots - 1)
	if size <= 0 {
		size = 1
	}

	input := make([]byte, size)
	if _, err := rand.Read(input); err != nil {
		return nil
	}

	return &types.Transaction{
		From:     addr,
		Nonce:    nonce,
		Value:    big.NewInt(1),
		GasPrice: price,
		Gas:      validGasLimit,
		Input:    input,
	}
}

// returns a new txpool with default test config
func newTestPool(mockStore ...store) (*TxPool, error) {
	return newTestPoolWithSlots(defaultMaxSlots, mockStore...)
}

func newTestPoolWithSlots(maxSlots uint64, mockStore ...store) (*TxPool, error) {
	var storeToUse store
	if len(mockStore) != 0 {
		storeToUse = mockStore[0]
	} else {
		storeToUse = defaultMockStore{
			DefaultHeader: mockHeader,
		}
	}

	return NewTxPool(
		hclog.NewNullLogger(),
		forks.At(0),
		storeToUse,
		nil,
		nil,
		nilMetrics,
		&Config{
			PriceLimit:            defaultPriceLimit,
			MaxSlots:              maxSlots,
			Sealing:               false,
			PruneTickSeconds:      DefaultPruneTickSeconds,
			PromoteOutdateSeconds: DefaultPromoteOutdateSeconds,
			ClippingMemory: ClippingMemory{
				TickSeconds: DefaultClippingTickSeconds,
				Threshold:   DefaultClippingMemoryThreshold,
			},
		},
	)
}

type accountState struct {
	enqueued, promoted, nextNonce uint64
}

type result struct {
	accounts map[types.Address]accountState
	slots    uint64
}

/* Single account cases (unit tests) */

func TestAddTxErrors(t *testing.T) {
	poolSigner := crypto.NewEIP155Signer(100)

	// Generate a private key and address
	defaultKey, defaultAddr := tests.GenerateKeyAndAddr(t)

	setupPool := func() *TxPool {
		pool, err := newTestPool()
		if err != nil {
			t.Fatalf("cannot create txpool - err: %v\n", err)
		}

		pool.SetSigner(poolSigner)

		return pool
	}

	signTx := func(transaction *types.Transaction) *types.Transaction {
		signedTx, signErr := poolSigner.SignTx(transaction, defaultKey)
		if signErr != nil {
			t.Fatalf("Unable to sign transaction, %v", signErr)
		}

		return signedTx
	}

	t.Run("ErrNegativeValue", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(defaultAddr, 0, 1)
		tx.Value = big.NewInt(-5)

		assert.ErrorIs(t,
			pool.addTx(local, signTx(tx)),
			ErrNegativeValue,
		)
	})

	t.Run("ErrBlockLimitExceeded", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(defaultAddr, 0, 1)
		tx.Value = big.NewInt(1)
		tx.Gas = 10000000000001

		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrBlockLimitExceeded,
		)
	})

	t.Run("ErrExtractSignature", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(defaultAddr, 0, 1)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrExtractSignature,
		)
	})

	t.Run("ErrInvalidSender", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(addr1, 0, 1)

		// Sign with a private key that corresponds
		// to a different address
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrInvalidSender,
		)
	})

	t.Run("ErrUnderpriced", func(t *testing.T) {
		pool := setupPool()
		pool.priceLimit = 1000000

		tx := newTx(defaultAddr, 0, 1) // gasPrice == 1
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrUnderpriced,
		)
	})

	t.Run("ErrInvalidAccountState", func(t *testing.T) {
		pool := setupPool()
		pool.store = faultyMockStore{}

		// nonce is 1000000 so ErrNonceTooLow
		// doesn't get triggered
		tx := newTx(defaultAddr, 1000000, 1)
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrInvalidAccountState,
		)
	})

	t.Run("ErrTxPoolOverflow", func(t *testing.T) {
		pool := setupPool()

		// fill the pool
		pool.gauge.increase(defaultMaxSlots)

		tx := newTx(defaultAddr, 0, 1)
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrTxPoolOverflow,
		)
	})

	t.Run("ErrIntrinsicGas", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(defaultAddr, 0, 1)
		tx.Gas = 1
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrIntrinsicGas,
		)
	})

	t.Run("ErrAlreadyKnown", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(defaultAddr, 0, 1)
		tx = signTx(tx)

		// send the tx beforehand
		go func() {
			err := pool.addTx(local, tx)
			assert.NoError(t, err)
		}()

		go pool.handleEnqueueRequest(<-pool.enqueueReqCh)
		<-pool.promoteReqCh

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrAlreadyKnown,
		)
	})

	t.Run("ErrOversizedData", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(defaultAddr, 0, 1)

		// set oversized Input field
		data := make([]byte, 989898)
		_, err := rand.Read(data)
		assert.NoError(t, err)

		tx.Input = data
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrOversizedData,
		)
	})

	t.Run("ErrNonceTooLow", func(t *testing.T) {
		pool := setupPool()

		// faultyMockStore.GetNonce() == 99999
		pool.store = faultyMockStore{}
		tx := newTx(defaultAddr, 0, 1)
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrNonceTooLow,
		)
	})

	t.Run("ErrInsufficientFunds", func(t *testing.T) {
		pool := setupPool()

		tx := newTx(defaultAddr, 0, 1)
		tx.GasPrice.SetUint64(1000000000000)
		tx = signTx(tx)

		assert.ErrorIs(t,
			pool.addTx(local, tx),
			ErrInsufficientFunds,
		)
	})
}

func TestAddGossipTx(t *testing.T) {
	key, sender := tests.GenerateKeyAndAddr(t)
	signer := crypto.NewEIP155Signer(uint64(100))
	tx := newTx(types.ZeroAddress, 1, 1)

	t.Run("node is a validator", func(t *testing.T) {
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(signer)

		pool.sealing = true

		signedTx, err := signer.SignTx(tx, key)
		if err != nil {
			t.Fatalf("cannot sign transction - err: %v", err)
		}

		// send tx
		go func() {
			protoTx := &proto.Txn{
				Raw: &any.Any{
					Value: signedTx.MarshalRLP(),
				},
			}
			pool.addGossipTx(protoTx)
		}()
		pool.handleEnqueueRequest(<-pool.enqueueReqCh)

		assert.Equal(t, uint64(1), pool.accounts.get(sender).enqueued.length())
	})

	t.Run("node is a non validator", func(t *testing.T) {
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(signer)

		pool.sealing = false

		pool.createAccountOnce(sender)

		signedTx, err := signer.SignTx(tx, key)
		if err != nil {
			t.Fatalf("cannot sign transction - err: %v", err)
		}

		// send tx
		protoTx := &proto.Txn{
			Raw: &any.Any{
				Value: signedTx.MarshalRLP(),
			},
		}
		pool.addGossipTx(protoTx)

		assert.Equal(t, uint64(0), pool.accounts.get(sender).enqueued.length())
	})
}

func TestDropKnownGossipTx(t *testing.T) {
	t.Parallel()

	pool, err := newTestPool()
	assert.NoError(t, err)
	pool.SetSigner(&mockSigner{})

	tx := newTx(addr1, 1, 1)

	// send tx as local
	go func() {
		assert.NoError(t, pool.addTx(local, tx))
	}()
	<-pool.enqueueReqCh

	_, exists := pool.index.get(tx.Hash)
	assert.True(t, exists)

	// send tx as gossip (will be discarded)
	assert.ErrorIs(t,
		pool.addTx(gossip, tx),
		ErrAlreadyKnown,
	)
}

func TestAddGossipTx_ShouldNotCrash(t *testing.T) {
	pool, err := newTestPool()
	assert.NoError(t, err)
	pool.SetSigner(&mockSigner{})

	assert.NotPanics(t, func() {
		// type assertion
		pool.addGossipTx(&types.Block{Header: &types.Header{Number: 10}})
	})

	assert.Nil(t, pool.accounts.get(addr1), "addr in txpool should be nil")

	assert.NotPanics(t, func() {
		pool.addGossipTx(createNilRawTxn())
		pool.addGossipTx(createNilRawDataTxn())
	})

	assert.Nil(t, pool.accounts.get(addr1), "addr in txpool should be nil")
}

func createNilRawTxn() *proto.Txn {
	return &proto.Txn{
		Raw: nil,
	}
}

func createNilRawDataTxn() *proto.Txn {
	return &proto.Txn{
		Raw: &anypb.Any{
			Value: nil,
		},
	}
}

func TestAddHandler(t *testing.T) {
	t.Run("enqueue new tx with higher nonce", func(t *testing.T) {
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		// send higher nonce tx
		go func() {
			err := pool.addTx(local, newTx(addr1, 10, 1)) // 10 > 0
			assert.NoError(t, err)
		}()
		pool.handleEnqueueRequest(<-pool.enqueueReqCh)

		assert.Equal(t, uint64(1), pool.gauge.read())
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).enqueued.length())
	})

	t.Run("reject new tx with low nonce", func(t *testing.T) {
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		// setup prestate
		acc := pool.createAccountOnce(addr1)
		acc.setNonce(20)

		// send tx
		go func() {
			err := pool.addTx(local, newTx(addr1, 10, 1)) // 10 < 20
			assert.NoError(t, err)
		}()
		pool.handleEnqueueRequest(<-pool.enqueueReqCh)

		assert.Equal(t, uint64(0), pool.gauge.read())
		assert.Equal(t, uint64(0), pool.accounts.get(addr1).enqueued.length())
	})

	t.Run("signal promotion for new tx with expected nonce", func(t *testing.T) {
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		// send tx
		go func() {
			err := pool.addTx(local, newTx(addr1, 0, 1)) // 0 == 0
			assert.NoError(t, err)
		}()
		go pool.handleEnqueueRequest(<-pool.enqueueReqCh)

		// catch pending promotion
		<-pool.promoteReqCh

		assert.Equal(t, uint64(1), pool.gauge.read())
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())
	})
}

func TestPromoteHandler(t *testing.T) {
	t.Run("nothing to promote", func(t *testing.T) {
		/* This test demonstrates that if some promotion handler
		got its job done by a previous one, it will not perform any logic
		by doing an early return. */
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		// fake a promotion signal
		signalPromotion := func() {
			pool.promoteReqCh <- promoteRequest{account: addr1}
		}

		// fresh account (queues are empty)
		acc := pool.createAccountOnce(addr1)
		acc.setNonce(7)

		// fake a promotion
		go signalPromotion()
		pool.handlePromoteRequest(<-pool.promoteReqCh)
		assert.Equal(t, uint64(0), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())

		// enqueue higher nonce tx
		go func() {
			err := pool.addTx(local, newTx(addr1, 10, 1))
			assert.NoError(t, err)
		}()
		pool.handleEnqueueRequest(<-pool.enqueueReqCh)
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())

		// fake a promotion
		go signalPromotion()
		pool.handlePromoteRequest(<-pool.promoteReqCh)
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())
	})

	t.Run("promote one tx", func(t *testing.T) {
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		go func() {
			err := pool.addTx(local, newTx(addr1, 0, 1))
			assert.NoError(t, err)
		}()
		go pool.handleEnqueueRequest(<-pool.enqueueReqCh)

		// tx enqueued -> promotion signaled
		pool.handlePromoteRequest(<-pool.promoteReqCh)

		assert.Equal(t, uint64(1), pool.gauge.read())
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).getNonce())

		assert.Equal(t, uint64(0), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).promoted.length())
	})

	t.Run("promote several txs", func(t *testing.T) {
		/* This example illustrates the flexibility of the handlers:
		One promotion handler can be executed at any time after it
		was invoked (when the runtime decides), resulting in promotion
		of several enqueued txs. */
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		// send the first (expected) tx -> signals promotion
		go func() {
			err := pool.addTx(local, newTx(addr1, 0, 1)) // 0 == 0
			assert.NoError(t, err)
		}()
		go pool.handleEnqueueRequest(<-pool.enqueueReqCh)

		// save the promotion handler
		req := <-pool.promoteReqCh

		// send the remaining txs (all will be enqueued)
		for nonce := uint64(1); nonce < 10; nonce++ {
			go func() {
				err := pool.addTx(local, newTx(addr1, nonce, 1))
				assert.NoError(t, err)
			}()
			pool.handleEnqueueRequest(<-pool.enqueueReqCh)
		}

		// verify all 10 are enqueued
		assert.Equal(t, uint64(10), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())

		// execute the promotion handler
		pool.handlePromoteRequest(req)

		assert.Equal(t, uint64(10), pool.gauge.read())
		assert.Equal(t, uint64(10), pool.accounts.get(addr1).getNonce())

		assert.Equal(t, uint64(0), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(10), pool.accounts.get(addr1).promoted.length())
	})

	t.Run("one tx -> one promotion", func(t *testing.T) {
		/* In this scenario, each received tx will be instantly promoted.
		All txs are sent in the order of expected nonce. */
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		for nonce := uint64(0); nonce < 20; nonce++ {
			go func(nonce uint64) {
				err := pool.addTx(local, newTx(addr1, nonce, 1))
				assert.NoError(t, err)
			}(nonce)
			go pool.handleEnqueueRequest(<-pool.enqueueReqCh)
			pool.handlePromoteRequest(<-pool.promoteReqCh)
		}

		assert.Equal(t, uint64(20), pool.gauge.read())
		assert.Equal(t, uint64(20), pool.accounts.get(addr1).getNonce())

		assert.Equal(t, uint64(0), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(20), pool.accounts.get(addr1).promoted.length())
	})

	t.Run("Two txs -> one promoted, the lower nonce dropped", func(t *testing.T) {
		/* This example illustrates the fault tolerance of the promoting handler:
		The lower nonce transaction should be dropped when it is accidentally inserted
		into the enqueued queue. It should not block any promotable transactions. */
		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})
		const promotableNonce = uint64(5)

		// event sub
		subscription := pool.eventManager.subscribe([]proto.EventType{
			proto.EventType_PROMOTED,
			proto.EventType_PRUNED_ENQUEUED,
		})

		// Prepare the bug test case. It is a bug, but sometimes it happeds.
		// set account nonce large to make the first tx lower nonce
		acc := pool.createAccountOnce(addr1)
		acc.setNonce(promotableNonce)
		// lower nonce tx
		lowerNonceTx := newTx(addr1, 0, 1)
		// enqueue it by hand
		acc.enqueued.push(lowerNonceTx)
		pool.gauge.increase(1)
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).enqueued.length())

		go func() {
			// promotable
			err = pool.addTx(local, newTx(addr1, promotableNonce, 1))
			assert.NoError(t, err)
		}()
		// enqueue
		go pool.handleEnqueueRequest(<-pool.enqueueReqCh)
		// promote
		go pool.handlePromoteRequest(<-pool.promoteReqCh)

		// waiting for the promoted event
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		// 1 dropped, 1 promoted.
		events := waitForEvents(ctx, subscription, 2)

		var prunedCount, promotedCount int
		// count events
		for _, ev := range events {
			switch ev.Type {
			case proto.EventType_PROMOTED:
				promotedCount++
			case proto.EventType_PRUNED_ENQUEUED:
				prunedCount++
			}
		}

		// assert
		assert.Equal(t, 1, prunedCount)
		assert.Equal(t, 1, promotedCount)

		assert.Equal(t, uint64(1), pool.gauge.read())
		assert.Equal(t, promotableNonce+1, pool.accounts.get(addr1).getNonce())

		assert.Equal(t, uint64(0), pool.accounts.get(addr1).enqueued.length())
		assert.Equal(t, uint64(1), pool.accounts.get(addr1).promoted.length())
	})
}

func TestResetAccount(t *testing.T) {
	t.Parallel()

	t.Run("reset promoted", func(t *testing.T) {
		t.Parallel()

		testCases := []*struct {
			name     string
			txs      []*types.Transaction
			newNonce uint64
			expected result
		}{
			{
				name: "prune all txs with low nonce",
				txs: []*types.Transaction{
					newTx(addr1, 0, 1),
					newTx(addr1, 1, 1),
					newTx(addr1, 2, 1),
					newTx(addr1, 3, 1),
					newTx(addr1, 4, 1),
				},
				newNonce: 5,
				expected: result{
					slots: 0,
					accounts: map[types.Address]accountState{
						addr1: {
							promoted: 0,
						},
					},
				},
			},
			{
				name: "no low nonce txs to prune",
				txs: []*types.Transaction{
					newTx(addr1, 2, 1),
					newTx(addr1, 3, 1),
					newTx(addr1, 4, 1),
				},
				newNonce: 1,
				expected: result{
					slots: 3,
					accounts: map[types.Address]accountState{
						addr1: {
							promoted: 3,
						},
					},
				},
			},
			{
				name: "prune some txs with low nonce",
				txs: []*types.Transaction{
					newTx(addr1, 7, 1),
					newTx(addr1, 8, 1),
					newTx(addr1, 9, 1),
				},
				newNonce: 8,
				expected: result{
					slots: 2,
					accounts: map[types.Address]accountState{
						addr1: {
							promoted: 2,
						},
					},
				},
			},
		}
		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				pool, err := newTestPool()
				assert.NoError(t, err)
				pool.SetSigner(&mockSigner{})

				// setup prestate
				acc := pool.createAccountOnce(addr1)
				acc.setNonce(test.txs[0].Nonce)

				go func() {
					err := pool.addTx(local, test.txs[0])
					assert.NoError(t, err)
				}()
				go pool.handleEnqueueRequest(<-pool.enqueueReqCh)

				// save the promotion
				req := <-pool.promoteReqCh

				// enqueue remaining
				for i, tx := range test.txs {
					if i == 0 {
						// first was handled
						continue
					}
					go func(tx *types.Transaction) {
						err := pool.addTx(local, tx)
						assert.NoError(t, err)
					}(tx)
					pool.handleEnqueueRequest(<-pool.enqueueReqCh)
				}

				pool.handlePromoteRequest(req)
				assert.Equal(t, uint64(0), pool.accounts.get(addr1).enqueued.length())
				assert.Equal(t, uint64(len(test.txs)), pool.accounts.get(addr1).promoted.length())

				pool.resetAccounts(map[types.Address]uint64{
					addr1: test.newNonce,
				})

				assert.Equal(t, test.expected.slots, pool.gauge.read())
				assert.Equal(t, // enqueued
					test.expected.accounts[addr1].enqueued,
					pool.accounts.get(addr1).enqueued.length())
				assert.Equal(t, // promoted
					test.expected.accounts[addr1].promoted,
					pool.accounts.get(addr1).promoted.length())
			})
		}
	})

	t.Run("reset enqueued", func(t *testing.T) {
		t.Parallel()

		testCases := []*struct {
			name     string
			txs      []*types.Transaction
			newNonce uint64
			expected result
			signal   bool // flag indicating whether reset will cause a promotion
		}{
			{
				name: "prune all txs with low nonce",
				txs: []*types.Transaction{
					newTx(addr1, 5, 1),
					newTx(addr1, 6, 1),
					newTx(addr1, 7, 1),
					newTx(addr1, 8, 1),
				},
				newNonce: 10,
				expected: result{
					slots: 0,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 0,
						},
					},
				},
			},
			{
				name: "no low nonce txs to prune",
				txs: []*types.Transaction{
					newTx(addr1, 2, 1),
					newTx(addr1, 3, 1),
					newTx(addr1, 4, 1),
				},
				newNonce: 1,
				expected: result{
					slots: 3,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 3,
						},
					},
				},
			},
			{
				name: "prune some txs with low nonce",
				txs: []*types.Transaction{
					newTx(addr1, 4, 1),
					newTx(addr1, 5, 1),
					newTx(addr1, 8, 1),
					newTx(addr1, 9, 1),
				},
				newNonce: 6,
				expected: result{
					slots: 2,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 2,
						},
					},
				},
			},
			{
				name:   "pruning low nonce signals promotion",
				signal: true,
				txs: []*types.Transaction{
					newTx(addr1, 8, 1),
					newTx(addr1, 9, 1),
					newTx(addr1, 10, 1),
				},
				newNonce: 9,
				expected: result{
					slots: 2,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 0,
							promoted: 2,
						},
					},
				},
			},
		}

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				pool, err := newTestPool()
				assert.NoError(t, err)
				pool.SetSigner(&mockSigner{})

				// setup prestate
				for _, tx := range test.txs {
					go func(tx *types.Transaction) {
						err := pool.addTx(local, tx)
						assert.NoError(t, err)
					}(tx)
					pool.handleEnqueueRequest(<-pool.enqueueReqCh)
				}

				assert.Equal(t, uint64(len(test.txs)), pool.accounts.get(addr1).enqueued.length())
				assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())

				if test.signal {
					go pool.resetAccounts(map[types.Address]uint64{
						addr1: test.newNonce,
					})
					pool.handlePromoteRequest(<-pool.promoteReqCh)
				} else {
					pool.resetAccounts(map[types.Address]uint64{
						addr1: test.newNonce,
					})
				}

				assert.Equal(t, test.expected.slots, pool.gauge.read())
				assert.Equal(t, // enqueued
					test.expected.accounts[addr1].enqueued,
					pool.accounts.get(addr1).enqueued.length())
				assert.Equal(t, // promoted
					test.expected.accounts[addr1].promoted,
					pool.accounts.get(addr1).promoted.length())
			})
		}
	})

	t.Run("reset enqueued and promoted", func(t *testing.T) {
		t.Parallel()

		testCases := []*struct {
			name     string
			txs      []*types.Transaction
			newNonce uint64
			expected result
			signal   bool // flag indicating whether reset will cause a promotion
		}{
			{
				name: "prune all txs with low nonce",
				txs: []*types.Transaction{
					// promoted
					newTx(addr1, 0, 1),
					newTx(addr1, 1, 1),
					newTx(addr1, 2, 1),
					newTx(addr1, 3, 1),
					// enqueued
					newTx(addr1, 5, 1),
					newTx(addr1, 6, 1),
					newTx(addr1, 8, 1),
				},
				newNonce: 10,
				expected: result{
					slots: 0,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 0,
							promoted: 0,
						},
					},
				},
			},
			{
				name: "no low nonce txs to prune",
				txs: []*types.Transaction{
					// promoted
					newTx(addr1, 5, 1),
					newTx(addr1, 6, 1),
					// enqueued
					newTx(addr1, 9, 1),
					newTx(addr1, 10, 1),
				},
				newNonce: 3,
				expected: result{
					slots: 4,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 2,
							promoted: 2,
						},
					},
				},
			},
			{
				name: "prune all promoted and 1 enqueued",
				txs: []*types.Transaction{
					// promoted
					newTx(addr1, 1, 1),
					newTx(addr1, 2, 1),
					newTx(addr1, 3, 1),
					// enqueued
					newTx(addr1, 5, 1),
					newTx(addr1, 8, 1),
					newTx(addr1, 9, 1),
				},
				newNonce: 6,
				expected: result{
					slots: 2,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 2,
							promoted: 0,
						},
					},
				},
			},
			{
				name:   "prune signals promotion",
				signal: true,
				txs: []*types.Transaction{
					// promoted
					newTx(addr1, 2, 1),
					newTx(addr1, 3, 1),
					newTx(addr1, 4, 1),
					newTx(addr1, 5, 1),
					// enqueued
					newTx(addr1, 8, 1),
					newTx(addr1, 9, 1),
					newTx(addr1, 10, 1),
				},
				newNonce: 8,
				expected: result{
					slots: 3,
					accounts: map[types.Address]accountState{
						addr1: {
							enqueued: 0,
							promoted: 3,
						},
					},
				},
			},
		}

		for _, test := range testCases {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				pool, err := newTestPool()
				assert.NoError(t, err)
				pool.SetSigner(&mockSigner{})

				// setup prestate
				acc := pool.createAccountOnce(addr1)
				acc.setNonce(test.txs[0].Nonce)

				go func() {
					err := pool.addTx(local, test.txs[0])
					assert.NoError(t, err)
				}()
				go pool.handleEnqueueRequest(<-pool.enqueueReqCh)

				// save the promotion
				req := <-pool.promoteReqCh

				// enqueue remaining
				for i, tx := range test.txs {
					if i == 0 {
						// first was handled
						continue
					}
					go func(tx *types.Transaction) {
						err := pool.addTx(local, tx)
						assert.NoError(t, err)
					}(tx)
					pool.handleEnqueueRequest(<-pool.enqueueReqCh)
				}

				pool.handlePromoteRequest(req)

				if test.signal {
					go pool.resetAccounts(map[types.Address]uint64{
						addr1: test.newNonce,
					})
					pool.handlePromoteRequest(<-pool.promoteReqCh)
				} else {
					pool.resetAccounts(map[types.Address]uint64{
						addr1: test.newNonce,
					})
				}

				assert.Equal(t, test.expected.slots, pool.gauge.read())
				assert.Equal(t, // enqueued
					test.expected.accounts[addr1].enqueued,
					pool.accounts.get(addr1).enqueued.length())
				assert.Equal(t, // promoted
					test.expected.accounts[addr1].promoted,
					pool.accounts.get(addr1).promoted.length())
			})
		}
	})
}

func TestPop(t *testing.T) {
	pool, err := newTestPool()
	assert.NoError(t, err)
	pool.SetSigner(&mockSigner{})

	// send 1 tx and promote it
	go func() {
		err := pool.addTx(local, newTx(addr1, 0, 1))
		assert.NoError(t, err)
	}()
	go pool.handleEnqueueRequest(<-pool.enqueueReqCh)
	pool.handlePromoteRequest(<-pool.promoteReqCh)

	assert.Equal(t, uint64(1), pool.gauge.read())
	assert.Equal(t, uint64(1), pool.accounts.get(addr1).promoted.length())

	// pop the tx
	pool.Prepare()
	tx := pool.Pop()
	pool.RemoveExecuted(tx)

	assert.Equal(t, uint64(0), pool.gauge.read())
	assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())
}

func TestDrop(t *testing.T) {
	pool, err := newTestPool()
	assert.NoError(t, err)
	pool.SetSigner(&mockSigner{})

	// send 1 tx and promote it
	go func() {
		err := pool.addTx(local, newTx(addr1, 0, 1))
		assert.NoError(t, err)
	}()
	go pool.handleEnqueueRequest(<-pool.enqueueReqCh)
	pool.handlePromoteRequest(<-pool.promoteReqCh)

	assert.Equal(t, uint64(1), pool.gauge.read())
	assert.Equal(t, uint64(1), pool.accounts.get(addr1).getNonce())
	assert.Equal(t, uint64(1), pool.accounts.get(addr1).promoted.length())

	// pop the tx
	pool.Prepare()
	tx := pool.Pop()
	pool.Drop(tx)

	assert.Equal(t, uint64(0), pool.gauge.read())
	assert.Equal(t, uint64(0), pool.accounts.get(addr1).getNonce())
	assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())
}

func TestDrop_RecoverRightNonce(t *testing.T) {
	pool, err := newTestPool()
	assert.NoError(t, err)
	pool.SetSigner(&mockSigner{})

	const maxTxLength = 2

	// send txs
	go func() {
		for i := 0; i < maxTxLength; i++ {
			err := pool.addTx(local, newTx(addr1, uint64(i), 1))
			assert.NoError(t, err)
		}
	}()
	// enqueue them
	go func() {
		for i := 0; i < maxTxLength; i++ {
			pool.handleEnqueueRequest(<-pool.enqueueReqCh)
		}
	}()
	// promote them
	go func() {
		for i := 0; i < maxTxLength; i++ {
			pool.handlePromoteRequest(<-pool.promoteReqCh)
		}
	}()

	// waiting for the promoted event
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	nonce, err := tests.RetryUntilTimeout(ctx, func() (interface{}, bool) {
		account1 := pool.accounts.get(addr1)
		if account1 == nil {
			return nil, true // retry
		}

		nonce := account1.getNonce()
		if nonce < maxTxLength {
			return nil, true // retry
		}

		return nonce, false
	})

	assert.NoError(t, err)
	assert.Equal(t, uint64(2), nonce)
	assert.Equal(t, uint64(2), pool.gauge.read())
	assert.Equal(t, uint64(2), pool.accounts.get(addr1).getNonce())
	assert.Equal(t, uint64(2), pool.accounts.get(addr1).promoted.length())

	// pop the tx
	pool.Prepare()
	tx := pool.Pop()
	pool.Drop(tx)

	assert.Equal(t, uint64(0), pool.gauge.read())
	assert.Equal(t, uint64(0), pool.accounts.get(addr1).getNonce())
	assert.Equal(t, uint64(0), pool.accounts.get(addr1).promoted.length())
}

func TestTxpool_PruneStaleAccounts(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name            string
		futureTx        *types.Transaction
		lastPromoted    time.Time
		expectedTxCount uint64
		expectedGauge   uint64
	}{
		{
			"prune stale tx",
			newTx(addr1, 3, 1),
			time.Now().Add(-time.Second * DefaultPromoteOutdateSeconds),
			0,
			0,
		},
		{
			"no stale tx to prune",
			newTx(addr1, 5, 1),
			time.Now().Add(-5 * time.Second),
			1,
			1,
		},
	}

	for _, test := range testTable {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pool, err := newTestPool()
			assert.NoError(t, err)
			pool.SetSigner(&mockSigner{})

			go func() {
				// add a future nonce tx
				err := pool.addTx(local, test.futureTx)
				assert.NoError(t, err)
			}()
			pool.handleEnqueueRequest(<-pool.enqueueReqCh)

			acc := pool.accounts.get(addr1)
			acc.lastPromoted = test.lastPromoted

			assert.Equal(t, uint64(1), acc.enqueued.length())
			assert.Equal(t, uint64(1), pool.gauge.read())

			//	pretend ticker ticks and triggers the pruning cycle
			pool.pruneStaleAccounts()

			//	enqueued tx is removed
			assert.Equal(t, test.expectedTxCount, acc.enqueued.length())
			assert.Equal(t, test.expectedGauge, pool.gauge.read())
		})
	}
}

func BenchmarkPruneStaleAccounts1KAccounts(b *testing.B)   { benchmarkPruneStaleAccounts(b, 1000) }
func BenchmarkPruneStaleAccounts10KAccounts(b *testing.B)  { benchmarkPruneStaleAccounts(b, 10000) }
func BenchmarkPruneStaleAccounts100KAccounts(b *testing.B) { benchmarkPruneStaleAccounts(b, 100000) }

func benchmarkPruneStaleAccounts(b *testing.B, accountSize int) {
	b.Helper()

	pool, err := newTestPoolWithSlots(uint64(accountSize + 1))
	assert.NoError(b, err)

	pool.SetSigner(&mockSigner{})
	pool.pruneTick = time.Second  // prune check on every second
	pool.clippingTick = time.Hour // 'disable' pool memory clipping

	pool.Start()
	defer pool.Close()

	var (
		addresses        = make([]types.Address, accountSize)
		lastPromotedTime = time.Now()
	)

	for i := 0; i < accountSize; i++ {
		addresses[i] = types.StringToAddress("0x" + strconv.FormatInt(int64(1024+i), 16))
		addr := addresses[i]
		// add enough future tx
		err := pool.addTx(local, newTx(addr, uint64(10+i), 1))
		if !assert.NoError(b, err, "add tx failed") {
			b.FailNow()
		}
		// set the account lastPromoted to the same timestamp
		if !pool.accounts.exists(addr) {
			pool.createAccountOnce(addr)
		}

		pool.accounts.get(addr).lastPromoted = lastPromotedTime
	}

	// mark all transactions outdated
	pool.promoteOutdateDuration = time.Since(lastPromotedTime) - time.Millisecond

	// Reset the benchmark and measure pruning task
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// benchmark pruning task
		pool.pruneStaleAccounts()
	}

	promoted, enqueued := pool.GetTxs(true)
	assert.Len(b, promoted, 0)
	assert.Len(b, enqueued, 0)
}

func TestTxpool_ClipMemoryEater(t *testing.T) {
	testTable := []*struct {
		name            string
		txs             []*types.Transaction
		maxSlots        uint64
		memoryThreshold uint64
		expectedTxCount int
		expectedGauge   uint64
	}{
		{
			"clip max tx count account",
			[]*types.Transaction{
				newTx(addr1, 1, 1),
				newTx(addr2, 2, 1),
				newTx(addr2, 3, 1),
			},
			300,
			1,
			1,
			1,
		},
		{
			"clip random account when tx count equal",
			[]*types.Transaction{
				newTx(addr1, 1, 1),
				newTx(addr2, 2, 1),
				newTx(addr3, 3, 1),
			},
			300,
			1,
			2,
			2,
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			pool, err := newTestPoolWithSlots(test.maxSlots)
			assert.NoError(t, err)
			pool.SetSigner(&mockSigner{})
			// set the clipping memory threshold
			pool.clippingMemoryThreshold = test.memoryThreshold

			subscription := pool.eventManager.subscribe([]proto.EventType{proto.EventType_ENQUEUED})

			// add future nonce txs
			go func() {
				for _, tx := range test.txs {
					err := pool.addTx(local, tx)
					assert.NoError(t, err)
				}
			}()
			// handle enqueued
			go func() {
				for range test.txs {
					pool.handleEnqueueRequest(<-pool.enqueueReqCh)
				}
			}()

			// timeout context
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// subscription for events
			events := waitForEvents(ctx, subscription, len(test.txs))
			assert.Len(t, events, len(test.txs))

			// make the clipping manually
			pool.clipMemoryEater()

			// get enqueued tx
			_, enqueued := pool.GetTxs(true)
			txCount := 0
			//sum up enqueued tx count
			for _, accTxs := range enqueued {
				txCount += len(accTxs)
			}
			// assert enqueued tx count
			assert.Equal(t, txCount, test.expectedTxCount)

			// assert txpool gauge
			assert.Equal(t, test.expectedGauge, pool.gauge.read())
		})
	}
}

func BenchmarkClipMemoryEater1KAccounts(b *testing.B)   { benchmarkClipMemoryEater(b, 1000) }
func BenchmarkClipMemoryEater10KAccounts(b *testing.B)  { benchmarkClipMemoryEater(b, 10000) }
func BenchmarkClipMemoryEater100KAccounts(b *testing.B) { benchmarkClipMemoryEater(b, 100000) }

func benchmarkClipMemoryEater(b *testing.B, accountSize int) {
	b.Helper()

	slots := uint64(accountSize + 1)
	pool, err := newTestPoolWithSlots(slots)
	assert.NoError(b, err)

	pool.SetSigner(&mockSigner{})
	pool.pruneTick = time.Hour    // 'disable' prune check
	pool.clippingTick = time.Hour // 'disable' pool memory check
	// lowest clipping threshold, so every loop of benchmark might works
	pool.clippingMemoryThreshold = 1

	pool.Start()
	defer pool.Close()

	var addresses = make([]types.Address, accountSize)

	for i := 0; i < accountSize; i++ {
		addresses[i] = types.StringToAddress("0x" + strconv.FormatInt(int64(1024+i), 16))
		addr := addresses[i]
		// add enough future tx
		err := pool.addTx(local, newTx(addr, uint64(10+i), 1))
		if !assert.NoError(b, err, "add tx failed") {
			b.FailNow()
		}
	}

	// Reset the benchmark and measure clipping task
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// benchmark clipping task
		pool.clipMemoryEater()
	}
}

/* "Integrated" tests */

// The following tests ensure that the pool's inner event loop
// is handling requests correctly, meaning that we do not have
// to assume its role (like in previous unit tests) and
// perform dispatching/handling on our own

func waitForEvents(
	ctx context.Context,
	subscription *subscribeResult,
	count int,
) []*proto.TxPoolEvent {
	receivedEvents := make([]*proto.TxPoolEvent, 0)

	completed := false
	for !completed {
		select {
		case <-ctx.Done():
			completed = true
		case event := <-subscription.subscriptionChannel:
			receivedEvents = append(receivedEvents, event)

			if len(receivedEvents) == count {
				completed = true
			}
		}
	}

	return receivedEvents
}

type eoa struct {
	Address    types.Address
	PrivateKey *ecdsa.PrivateKey
}

func (e *eoa) create(t *testing.T) *eoa {
	t.Helper()

	e.PrivateKey, e.Address = tests.GenerateKeyAndAddr(t)

	return e
}

func (e *eoa) signTx(tx *types.Transaction, signer crypto.TxSigner) *types.Transaction {
	signedTx, err := signer.SignTx(tx, e.PrivateKey)
	if err != nil {
		panic("signTx failed")
	}

	return signedTx
}

var signerEIP155 = crypto.NewEIP155Signer(100)

func TestAddTxns(t *testing.T) {
	slotSize := uint64(1)

	testTable := []*struct {
		name   string
		numTxs uint64
	}{
		{
			"send 100 txns",
			100,
		},
		{
			"send 1k txns",
			1000,
		},
		{
			"send 10k txns",
			10000,
		},
		{
			"send 100k txns",
			100000,
		},
		{
			"send 1m txns",
			1000000,
		},
	}

	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			pool, err := newTestPoolWithSlots(test.numTxs * slotSize)

			assert.NoError(t, err)

			pool.SetSigner(&mockSigner{})

			pool.Start()
			defer pool.Close()

			subscription := pool.eventManager.subscribe([]proto.EventType{proto.EventType_PROMOTED})

			addr := types.Address{0x1}
			for nonce := uint64(0); nonce < test.numTxs; nonce++ {
				err := pool.addTx(local, newTx(addr, nonce, slotSize))
				assert.NoError(t, err)
			}

			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*20)
			defer cancelFunc()

			waitForEvents(ctx, subscription, int(test.numTxs))

			assert.Equal(t, test.numTxs, pool.accounts.get(addr).promoted.length())

			assert.Equal(t, test.numTxs*slotSize, pool.gauge.read())
		})
	}
}

func TestResetAccounts_Promoted(t *testing.T) {
	t.Parallel()

	var (
		eoa1 = new(eoa).create(t)
		eoa2 = new(eoa).create(t)
		eoa3 = new(eoa).create(t)
		eoa4 = new(eoa).create(t)

		addr1 = eoa1.Address
		addr2 = eoa2.Address
		addr3 = eoa3.Address
		addr4 = eoa4.Address
	)

	allTxs :=
		map[types.Address][]*types.Transaction{
			addr1: {
				eoa1.signTx(newTx(addr1, 0, 1), signerEIP155), // will be pruned
				eoa1.signTx(newTx(addr1, 1, 1), signerEIP155), // will be pruned
				eoa1.signTx(newTx(addr1, 2, 1), signerEIP155), // will be pruned
				eoa1.signTx(newTx(addr1, 3, 1), signerEIP155), // will be pruned
			},
			addr2: {
				eoa2.signTx(newTx(addr2, 0, 1), signerEIP155), // will be pruned
				eoa2.signTx(newTx(addr2, 1, 1), signerEIP155), // will be pruned
			},
			addr3: {
				eoa3.signTx(newTx(addr3, 0, 1), signerEIP155), // will be pruned
				eoa3.signTx(newTx(addr3, 1, 1), signerEIP155), // will be pruned
				eoa3.signTx(newTx(addr3, 2, 1), signerEIP155), // will be pruned
			},
			addr4: {
				//	all txs will be pruned
				eoa4.signTx(newTx(addr4, 0, 1), signerEIP155), // will be pruned
				eoa4.signTx(newTx(addr4, 1, 1), signerEIP155), // will be pruned
				eoa4.signTx(newTx(addr4, 2, 1), signerEIP155), // will be pruned
				eoa4.signTx(newTx(addr4, 3, 1), signerEIP155), // will be pruned
				eoa4.signTx(newTx(addr4, 4, 1), signerEIP155), // will be pruned
			},
		}

	newNonces := map[types.Address]uint64{
		addr1: 2,
		addr2: 1,
		addr3: 0,
		addr4: 5,
	}

	expected := result{
		accounts: map[types.Address]accountState{
			addr1: {promoted: 2},
			addr2: {promoted: 1},
			addr3: {promoted: 3},
			addr4: {promoted: 0},
		},
		slots: 2 + 1 + 3 + 0,
	}

	pool, err := newTestPool()
	assert.NoError(t, err)
	pool.SetSigner(signerEIP155)

	pool.Start()
	defer pool.Close()

	promotedSubscription := pool.eventManager.subscribe(
		[]proto.EventType{
			proto.EventType_PROMOTED,
		},
	)

	// setup prestate
	totalTx := 0

	for _, txs := range allTxs {
		for _, tx := range txs {
			totalTx++

			assert.NoError(t, pool.addTx(local, tx))
		}
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFn()

	// All txns should get added
	assert.Len(t, waitForEvents(ctx, promotedSubscription, totalTx), totalTx)
	pool.eventManager.cancelSubscription(promotedSubscription.subscriptionID)

	prunedSubscription := pool.eventManager.subscribe(
		[]proto.EventType{
			proto.EventType_PRUNED_PROMOTED,
		})

	pool.resetAccounts(newNonces)

	ctx, cancelFn = context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFn()

	assert.Len(t, waitForEvents(ctx, prunedSubscription, 8), 8)
	pool.eventManager.cancelSubscription(prunedSubscription.subscriptionID)

	assert.Equal(t, expected.slots, pool.gauge.read())

	for addr := range expected.accounts {
		assert.Equal(t, // enqueued
			expected.accounts[addr].enqueued,
			pool.accounts.get(addr).enqueued.length())

		assert.Equal(t, // promoted
			expected.accounts[addr].promoted,
			pool.accounts.get(addr).promoted.length())
	}
}

func TestResetAccounts_Enqueued(t *testing.T) {
	t.Parallel()

	commonAssert := func(accounts map[types.Address]accountState, pool *TxPool) {
		for addr := range accounts {
			assert.Equal(t, // enqueued
				accounts[addr].enqueued,
				pool.accounts.get(addr).enqueued.length())

			assert.Equal(t, // promoted
				accounts[addr].promoted,
				pool.accounts.get(addr).promoted.length())
		}
	}

	var (
		eoa1 = new(eoa).create(t)
		eoa2 = new(eoa).create(t)
		eoa3 = new(eoa).create(t)

		addr1 = eoa1.Address
		addr2 = eoa2.Address
		addr3 = eoa3.Address
	)

	t.Run("reset will promote", func(t *testing.T) {
		t.Parallel()

		allTxs := map[types.Address][]*types.Transaction{
			addr1: {
				eoa1.signTx(newTx(addr1, 3, 1), signerEIP155),
				eoa1.signTx(newTx(addr1, 4, 1), signerEIP155),
				eoa1.signTx(newTx(addr1, 5, 1), signerEIP155),
			},
			addr2: {
				eoa2.signTx(newTx(addr2, 2, 1), signerEIP155),
				eoa2.signTx(newTx(addr2, 3, 1), signerEIP155),
				eoa2.signTx(newTx(addr2, 4, 1), signerEIP155),
				eoa2.signTx(newTx(addr2, 5, 1), signerEIP155),
				eoa2.signTx(newTx(addr2, 6, 1), signerEIP155),
				eoa2.signTx(newTx(addr2, 7, 1), signerEIP155),
			},
			addr3: {
				eoa3.signTx(newTx(addr3, 7, 1), signerEIP155),
				eoa3.signTx(newTx(addr3, 8, 1), signerEIP155),
				eoa3.signTx(newTx(addr3, 9, 1), signerEIP155),
			},
		}
		newNonces := map[types.Address]uint64{
			addr1: 3,
			addr2: 4,
			addr3: 8,
		}
		expected := result{
			accounts: map[types.Address]accountState{
				addr1: {
					enqueued: 0,
					promoted: 3, // reset will promote
				},
				addr2: {
					enqueued: 0,
					promoted: 4,
				},
				addr3: {
					enqueued: 0,
					promoted: 2, // reset will promote
				},
			},
			slots: 3 + 4 + 2,
		}

		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(signerEIP155)

		pool.Start()
		defer pool.Close()

		enqueuedSubscription := pool.eventManager.subscribe(
			[]proto.EventType{
				proto.EventType_ENQUEUED,
			},
		)

		promotedSubscription := pool.eventManager.subscribe(
			[]proto.EventType{
				proto.EventType_PROMOTED,
			},
		)

		// setup prestate
		totalTx := 0
		expectedPromoted := uint64(0)
		for addr, txs := range allTxs {
			expectedPromoted += expected.accounts[addr].promoted
			for _, tx := range txs {
				totalTx++
				assert.NoError(t, pool.addTx(local, tx))
			}
		}

		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
		defer cancelFn()

		// All txns should get added
		assert.Len(t, waitForEvents(ctx, enqueuedSubscription, totalTx), totalTx)
		pool.eventManager.cancelSubscription(enqueuedSubscription.subscriptionID)

		pool.resetAccounts(newNonces)

		ctx, cancelFn = context.WithTimeout(context.Background(), time.Second*10)
		defer cancelFn()

		assert.Len(t, waitForEvents(ctx, promotedSubscription, int(expectedPromoted)), int(expectedPromoted))

		assert.Equal(t, expected.slots, pool.gauge.read())
		commonAssert(expected.accounts, pool)
	})

	t.Run("reset will not promote", func(t *testing.T) {
		t.Parallel()

		allTxs := map[types.Address][]*types.Transaction{
			addr1: {
				newTx(addr1, 1, 1),
				newTx(addr1, 2, 1),
				newTx(addr1, 3, 1),
				newTx(addr1, 4, 1),
			},
			addr2: {
				newTx(addr2, 3, 1),
				newTx(addr2, 4, 1),
				newTx(addr2, 5, 1),
				newTx(addr2, 6, 1),
			},
			addr3: {
				newTx(addr3, 7, 1),
				newTx(addr3, 8, 1),
				newTx(addr3, 9, 1),
			},
		}
		newNonces := map[types.Address]uint64{
			addr1: 5,
			addr2: 7,
			addr3: 12,
		}
		expected := result{
			accounts: map[types.Address]accountState{
				addr1: {
					enqueued: 0,
					promoted: 0, // reset will promote
				},
				addr2: {
					enqueued: 0,
					promoted: 0,
				},
				addr3: {
					enqueued: 0,
					promoted: 0, // reset will promote
				},
			},
			slots: 0 + 0 + 0,
		}

		pool, err := newTestPool()
		assert.NoError(t, err)
		pool.SetSigner(&mockSigner{})

		pool.Start()
		defer pool.Close()

		enqueuedSubscription := pool.eventManager.subscribe(
			[]proto.EventType{
				proto.EventType_ENQUEUED,
			},
		)

		// setup prestate
		expectedEnqueuedTx := 0
		expectedPromotedTx := uint64(0)
		for addr, txs := range allTxs {
			expectedPromotedTx += expected.accounts[addr].promoted
			for _, tx := range txs {
				expectedEnqueuedTx++
				assert.NoError(t, pool.addTx(local, tx))
			}
		}

		ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
		defer cancelFn()

		// All txns should get added
		assert.Len(t, waitForEvents(ctx, enqueuedSubscription, expectedEnqueuedTx), expectedEnqueuedTx)
		pool.eventManager.cancelSubscription(enqueuedSubscription.subscriptionID)

		pool.resetAccounts(newNonces)

		assert.Equal(t, expected.slots, pool.gauge.read())
		commonAssert(expected.accounts, pool)
	})
}

func TestExecutablesOrder(t *testing.T) {
	t.Parallel()

	newPricedTx := func(addr types.Address, nonce, gasPrice uint64) *types.Transaction {
		tx := newTx(addr, nonce, 1)
		tx.GasPrice.SetUint64(gasPrice)

		return tx
	}

	testCases := []*struct {
		name               string
		allTxs             map[types.Address][]*types.Transaction
		expectedPriceOrder []uint64
	}{
		{
			name: "case #1",
			allTxs: map[types.Address][]*types.Transaction{
				addr1: {
					newPricedTx(addr1, 0, 1),
				},
				addr2: {
					newPricedTx(addr2, 0, 2),
				},
				addr3: {
					newPricedTx(addr3, 0, 3),
				},
				addr4: {
					newPricedTx(addr4, 0, 4),
				},
				addr5: {
					newPricedTx(addr5, 0, 5),
				},
			},
			expectedPriceOrder: []uint64{
				5,
				4,
				3,
				2,
				1,
			},
		},
		{
			name: "case #2",
			allTxs: map[types.Address][]*types.Transaction{
				addr1: {
					newPricedTx(addr1, 0, 3),
					newPricedTx(addr1, 1, 3),
					newPricedTx(addr1, 2, 3),
				},
				addr2: {
					newPricedTx(addr2, 0, 2),
					newPricedTx(addr2, 1, 2),
					newPricedTx(addr2, 2, 2),
				},
				addr3: {
					newPricedTx(addr3, 0, 1),
					newPricedTx(addr3, 1, 1),
					newPricedTx(addr3, 2, 1),
				},
			},
			expectedPriceOrder: []uint64{
				3,
				3,
				3,
				2,
				2,
				2,
				1,
				1,
				1,
			},
		},
		{
			name: "case #3",
			allTxs: map[types.Address][]*types.Transaction{
				addr1: {
					newPricedTx(addr1, 0, 9),
					newPricedTx(addr1, 1, 5),
					newPricedTx(addr1, 2, 3),
				},
				addr2: {
					newPricedTx(addr2, 0, 9),
					newPricedTx(addr2, 1, 3),
					newPricedTx(addr2, 2, 1),
				},
			},
			expectedPriceOrder: []uint64{
				9,
				9,
				5,
				3,
				3,
				1,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pool, err := newTestPool()
			assert.NoError(t, err)
			pool.SetSigner(&mockSigner{})

			pool.Start()
			defer pool.Close()

			subscription := pool.eventManager.subscribe(
				[]proto.EventType{proto.EventType_PROMOTED},
			)

			expectedPromotedTx := 0
			for _, txs := range test.allTxs {
				for _, tx := range txs {
					expectedPromotedTx++
					// send all txs
					assert.NoError(t, pool.addTx(local, tx))
				}
			}

			ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
			defer cancelFn()

			// All txns should get added
			assert.Len(t, waitForEvents(ctx, subscription, expectedPromotedTx), expectedPromotedTx)
			assert.Equal(t, uint64(len(test.expectedPriceOrder)), pool.accounts.promoted())

			var successful []*types.Transaction
			for {
				tx := pool.Pop()
				if tx == nil {
					break
				}

				pool.RemoveExecuted(tx)
				successful = append(successful, tx)
			}

			// verify the highest priced transactions
			// were processed first
			for i, tx := range successful {
				assert.Equal(t, test.expectedPriceOrder[i], tx.GasPrice.Uint64())
			}
		})
	}
}

func TestDropAndRequeue(t *testing.T) {
	t.Parallel()

	type status int

	// Status of a transaction resulted
	// from a transition write attempt
	const (
		// if a tx is recoverable,
		// entire account is dropped,
		// the transactions behind it
		// is not executable
		recoverable status = iota

		// if a tx is unrecoverable,
		// entire account is dropped
		unrecoverable

		// if a tx is failed,
		// remove it only, but the other
		// txs would be demote if not
		// promotable
		failed

		ok
	)

	type statusTx struct {
		tx     *types.Transaction
		status status
	}

	type testCase struct {
		name               string
		allTxs             map[types.Address][]statusTx
		executableTxsCount uint64
		expected           result
	}

	commonAssert := func(test *testCase, pool *TxPool) {
		accounts := test.expected.accounts
		for addr := range accounts {
			assert.Equal(t, // nextNonce
				accounts[addr].nextNonce,
				pool.accounts.get(addr).getNonce(),
				fmt.Sprintf("%+v: %s nonce not equal", test.name, addr),
			)

			assert.Equal(t, // enqueued
				accounts[addr].enqueued,
				pool.accounts.get(addr).enqueued.length(),
				fmt.Sprintf("%+v: %s enqueued not equal", test.name, addr),
			)

			assert.Equal(t, // promoted
				accounts[addr].promoted,
				pool.accounts.get(addr).promoted.length(),
				fmt.Sprintf("%+v: %s promoted not equal", test.name, addr),
			)
		}
	}

	testCases := []*testCase{
		{
			name: "unrecoverable and recoverable all drops account",
			allTxs: map[types.Address][]statusTx{
				addr1: {
					{newTx(addr1, 0, 1), ok},
					{newTx(addr1, 1, 1), unrecoverable},
					{newTx(addr1, 2, 1), recoverable},
					{newTx(addr1, 3, 1), recoverable},
					{newTx(addr1, 4, 1), recoverable},
				},
				addr2: {
					{newTx(addr2, 9, 1), unrecoverable},
					{newTx(addr2, 10, 1), ok},
				},
				addr3: {
					{newTx(addr3, 5, 1), ok},
					{newTx(addr3, 6, 1), recoverable},
					{newTx(addr3, 7, 1), ok},
				},
			},
			executableTxsCount: 1 + 1, // addr1 + addr3
			expected: result{
				slots: 0, // all executed
				accounts: map[types.Address]accountState{
					addr1: {
						enqueued:  0,
						promoted:  0,
						nextNonce: 1,
					},
					addr2: {
						enqueued:  0,
						promoted:  0,
						nextNonce: 9,
					},
					addr3: {
						enqueued:  0,
						promoted:  0,
						nextNonce: 6,
					},
				},
			},
		},
		{
			name: "remove failed and re-enqueue",
			allTxs: map[types.Address][]statusTx{
				addr1: {
					{newTx(addr1, 0, 1), ok},
					{newTx(addr1, 1, 1), ok},
					{newTx(addr1, 2, 1), failed},
				},
				addr2: {
					{newTx(addr2, 9, 1), failed},
					{newTx(addr2, 10, 1), ok},
				},
			},
			executableTxsCount: 2, // all from addr1
			expected: result{
				slots: 1,
				accounts: map[types.Address]accountState{
					addr1: {
						enqueued:  0,
						promoted:  0,
						nextNonce: 2,
					},
					addr2: {
						enqueued:  1,
						promoted:  0,
						nextNonce: 9,
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// helper callback for transition errors
			status := func(tx *types.Transaction) (s status) {
				txs := test.allTxs[tx.From]
				for _, sTx := range txs {
					if tx.Nonce == sTx.tx.Nonce {
						s = sTx.status
					}
				}

				return
			}

			// create pool
			pool, err := newTestPool()
			assert.NoError(t, err)
			pool.SetSigner(&mockSigner{})

			pool.Start()
			defer pool.Close()

			promotionSubscription := pool.eventManager.subscribe([]proto.EventType{proto.EventType_PROMOTED})

			// setup prestate
			totalTx := 0
			expectedReenqueued := 0
			for addr, txs := range test.allTxs {
				// preset nonce so promotions can happen
				acc := pool.createAccountOnce(addr)
				acc.setNonce(txs[0].tx.Nonce)

				expectedReenqueued += int(test.expected.accounts[addr].enqueued)

				// send txs
				for _, sTx := range txs {
					totalTx++
					assert.NoError(t, pool.addTx(local, sTx.tx))
				}
			}

			// wait for all promotion
			ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
			defer cancelFn()

			// All txns should get added
			assert.Len(
				t,
				waitForEvents(ctx, promotionSubscription, totalTx),
				totalTx,
			)

			// re-enqueued or re-promote events
			reenqueueSubscription := pool.eventManager.subscribe([]proto.EventType{
				proto.EventType_ENQUEUED,
			})

			var executableTxsCount uint64
			func() {
				pool.Prepare()
				for {
					tx := pool.Pop()
					if tx == nil {
						break
					}

					switch status(tx) {
					case recoverable:
						pool.Drop(tx)
					case unrecoverable:
						pool.Drop(tx)
					case ok:
						executableTxsCount++
						pool.RemoveExecuted(tx)
					case failed:
						pool.RemoveFailed(tx)
					}
				}
			}()

			if expectedReenqueued > 0 {
				ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
				defer cancelFn()

				// All re-enqueued txs should get added
				assert.Len(
					t,
					waitForEvents(ctx, reenqueueSubscription, expectedReenqueued),
					expectedReenqueued,
				)
			}

			assert.Equal(t, test.executableTxsCount, executableTxsCount, "executable transaction count")
			assert.Equal(t, test.expected.slots, pool.gauge.read(), "gauge not equal")
			commonAssert(test, pool)
		})
	}
}

func TestGetTxs(t *testing.T) {
	t.Parallel()

	var (
		eoa1 = new(eoa).create(t)
		eoa2 = new(eoa).create(t)
		eoa3 = new(eoa).create(t)

		addr1 = eoa1.Address
		addr2 = eoa2.Address
		addr3 = eoa3.Address
	)

	testCases := []*struct {
		name             string
		allTxs           map[types.Address][]*types.Transaction
		expectedEnqueued map[types.Address][]*types.Transaction
		expectedPromoted map[types.Address][]*types.Transaction
	}{
		{
			name: "get promoted txs",
			allTxs: map[types.Address][]*types.Transaction{
				addr1: {
					eoa1.signTx(newTx(addr1, 0, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 1, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 2, 1), signerEIP155),
				},
				addr2: {
					eoa2.signTx(newTx(addr2, 0, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 1, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 2, 1), signerEIP155),
				},
				addr3: {
					eoa3.signTx(newTx(addr3, 0, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 1, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 2, 1), signerEIP155),
				},
			},
			expectedPromoted: map[types.Address][]*types.Transaction{
				addr1: {
					eoa1.signTx(newTx(addr1, 0, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 1, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 2, 1), signerEIP155),
				},
				addr2: {
					eoa2.signTx(newTx(addr2, 0, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 1, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 2, 1), signerEIP155),
				},
				addr3: {
					eoa3.signTx(newTx(addr3, 0, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 1, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 2, 1), signerEIP155),
				},
			},
		},
		{
			name: "get all txs",
			allTxs: map[types.Address][]*types.Transaction{
				addr1: {
					eoa1.signTx(newTx(addr1, 0, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 1, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 2, 1), signerEIP155),
					// enqueued
					eoa1.signTx(newTx(addr1, 10, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 11, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 12, 1), signerEIP155),
				},
				addr2: {
					eoa2.signTx(newTx(addr2, 0, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 1, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 2, 1), signerEIP155),
					// enqueued
					eoa2.signTx(newTx(addr2, 10, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 11, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 12, 1), signerEIP155),
				},
				addr3: {
					eoa3.signTx(newTx(addr3, 0, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 1, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 2, 1), signerEIP155),
					// enqueued
					eoa3.signTx(newTx(addr3, 10, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 11, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 12, 1), signerEIP155),
				},
			},
			expectedPromoted: map[types.Address][]*types.Transaction{
				addr1: {
					eoa1.signTx(newTx(addr1, 0, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 1, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 2, 1), signerEIP155),
				},
				addr2: {
					eoa2.signTx(newTx(addr2, 0, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 1, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 2, 1), signerEIP155),
				},
				addr3: {
					eoa3.signTx(newTx(addr3, 0, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 1, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 2, 1), signerEIP155),
				},
			},
			expectedEnqueued: map[types.Address][]*types.Transaction{
				addr1: {
					eoa1.signTx(newTx(addr1, 10, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 11, 1), signerEIP155),
					eoa1.signTx(newTx(addr1, 12, 1), signerEIP155),
				},
				addr2: {
					eoa2.signTx(newTx(addr2, 10, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 11, 1), signerEIP155),
					eoa2.signTx(newTx(addr2, 12, 1), signerEIP155),
				},
				addr3: {
					eoa3.signTx(newTx(addr3, 10, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 11, 1), signerEIP155),
					eoa3.signTx(newTx(addr3, 12, 1), signerEIP155),
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			find := func(
				tx *types.Transaction,
				all map[types.Address][]*types.Transaction,
			) bool {
				for _, txx := range all[tx.From] {
					if tx.Nonce == txx.Nonce {
						return true
					}
				}

				return false
			}

			pool, err := newTestPool()
			assert.NoError(t, err)
			pool.SetSigner(signerEIP155)

			pool.Start()
			defer pool.Close()

			promoteSubscription := pool.eventManager.subscribe(
				[]proto.EventType{
					proto.EventType_PROMOTED,
				},
			)

			enqueueSubscription := pool.eventManager.subscribe(
				[]proto.EventType{
					proto.EventType_ENQUEUED,
				},
			)

			// send txs
			expectedPromotedTx := 0
			for _, txs := range test.allTxs {
				nonce := uint64(0)
				promotable := uint64(0)
				for _, tx := range txs {
					// send all txs
					if tx.Nonce == nonce+promotable {
						promotable++
					}

					assert.NoError(t, pool.addTx(local, tx))
				}

				expectedPromotedTx += int(promotable)
			}

			ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
			defer cancelFn()

			// Wait for promoted transactions
			assert.Len(t, waitForEvents(ctx, promoteSubscription, expectedPromotedTx), expectedPromotedTx)

			// Wait for enqueued transactions, if any are present
			expectedEnqueuedTx := expectedPromotedTx - len(test.allTxs)

			if expectedEnqueuedTx > 0 {
				ctx, cancelFn = context.WithTimeout(context.Background(), time.Second*10)
				defer cancelFn()

				assert.Len(t, waitForEvents(ctx, enqueueSubscription, expectedEnqueuedTx), expectedEnqueuedTx)
			}

			allPromoted, allEnqueued := pool.GetTxs(true)

			// assert promoted
			for _, txs := range allPromoted {
				for _, tx := range txs {
					found := find(tx, test.expectedPromoted)
					assert.True(t, found)
				}
			}

			// assert enqueued
			for _, txs := range allEnqueued {
				for _, tx := range txs {
					found := find(tx, test.expectedEnqueued)
					assert.True(t, found)
				}
			}
		})
	}
}

func TestAddTx_ReplaceSameNonce(t *testing.T) {
	var (
		eoa  = new(eoa).create(t)
		addr = eoa.Address
		// price
		lowPrice  = big.NewInt(int64(defaultPriceLimit))
		midPrice  = big.NewInt(0).Mul(lowPrice, big.NewInt(2))
		highPrice = big.NewInt(0).Mul(lowPrice, big.NewInt(3))
		// normal txs
		tx0 = eoa.signTx(newTx(addr, 0, 1), signerEIP155)
		// same nonce txs
		tx1_1 = eoa.signTx(newPriceTx(addr, lowPrice, 1, 1), signerEIP155)
		tx1_2 = eoa.signTx(newPriceTx(addr, midPrice, 1, 1), signerEIP155)
		tx2_1 = eoa.signTx(newPriceTx(addr, lowPrice, 2, 1), signerEIP155)
		tx2_2 = eoa.signTx(newPriceTx(addr, midPrice, 2, 1), signerEIP155)
		tx3_1 = eoa.signTx(newPriceTx(addr, lowPrice, 3, 1), signerEIP155)
		tx3_2 = eoa.signTx(newPriceTx(addr, midPrice, 3, 1), signerEIP155)
		tx3_3 = eoa.signTx(newPriceTx(addr, highPrice, 3, 1), signerEIP155)
	)

	testCases := []struct {
		name                  string
		allTxs                []*types.Transaction
		expectedEnqueued      []*types.Transaction
		expectedPromoted      []*types.Transaction
		expectedPromotedCount int
		expectedReplacedCount int
	}{
		{
			name: "replace same nonce tx in enqueued list",
			allTxs: []*types.Transaction{
				tx1_1,
				tx1_2,
				tx2_1,
				tx2_2,
				tx3_1,
				tx3_2,
				tx3_3,
			},
			expectedEnqueued: []*types.Transaction{
				tx1_2,
				tx2_2,
				tx3_3,
			},
			expectedReplacedCount: 4,
		},
		{
			name: "replace same nonce tx in promoted list",
			allTxs: []*types.Transaction{
				tx0,
				tx1_1,
				tx1_2,
				tx2_1,
				tx2_2,
				tx3_1,
				tx3_2,
				tx3_3,
			},
			expectedPromoted: []*types.Transaction{
				tx0,
				tx1_2,
				tx2_2,
				tx3_3,
			},
			expectedPromotedCount: 8,
			expectedReplacedCount: 4,
		},
	}

	for _, test := range testCases {
		test := test

		t.Run(test.name, func(t *testing.T) {
			pool, err := newTestPool()
			assert.NoError(t, err)
			pool.SetSigner(signerEIP155)

			pool.Start()
			defer pool.Close()

			var eventSubscription = pool.eventManager.subscribe(
				[]proto.EventType{
					proto.EventType_PROMOTED,
					proto.EventType_REPLACED,
				},
			)

			// send txs in a goroutine, to avoid hanging event subscriptions.
			go func() {
				for _, tx := range test.allTxs {
					assert.NoError(t, pool.addTx(local, tx))
				}
			}()

			// Wait for promoted transactions
			ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
			defer cancelFn()

			expectedEventCount := test.expectedPromotedCount + test.expectedReplacedCount
			// the events might be emitted too soon in the rpc node, so do not rely on it
			waitForEvents(ctx, eventSubscription, expectedEventCount)

			allPromoted, allEnqueued := pool.GetTxs(true)

			// assert promoted
			assert.Equal(t, test.expectedPromoted, allPromoted[addr])

			// assert enqueued
			assert.Equal(t, test.expectedEnqueued, allEnqueued[addr])
		})
	}
}
