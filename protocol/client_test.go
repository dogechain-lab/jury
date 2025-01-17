package protocol

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dogechain-lab/dogechain/blockchain"
	"github.com/dogechain-lab/dogechain/network"
	"github.com/dogechain-lab/dogechain/network/event"
	"github.com/dogechain-lab/dogechain/network/grpc"
	"github.com/dogechain-lab/dogechain/protocol/proto"
	"github.com/dogechain-lab/dogechain/types"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

var (
	networkConfig = func(c *network.Config) {
		c.NoDiscover = true
	}
)

func newTestNetwork(t *testing.T) network.Server {
	t.Helper()

	srv, err := network.CreateServer(&network.CreateServerParams{
		ConfigCallback: networkConfig,
	})

	assert.NoError(t, err)

	return srv
}

func newTestSyncPeerClient(network network.Network, blockchain Blockchain) *syncPeerClient {
	ctx, cancel := context.WithCancel(context.Background())

	client := &syncPeerClient{
		logger:                 hclog.NewNullLogger(),
		network:                network,
		blockchain:             blockchain,
		selfID:                 network.AddrInfo().ID.String(),
		peerStatusUpdateCh:     make(chan *NoForkPeer, 1),
		peerConnectionUpdateCh: make(chan *event.PeerEvent, 1),
		isClosed:               atomic.NewBool(false),
		ctx:                    ctx,
		cancel:                 cancel,
	}

	// need to register protocol
	network.RegisterProtocol(_syncerV1, grpc.NewGrpcStream(context.TODO()))

	return client
}

func createTestSyncerService(t *testing.T, chain Blockchain) (*syncPeerService, network.Server) {
	t.Helper()

	srv := newTestNetwork(t)

	service := &syncPeerService{
		blockchain: chain,
		network:    srv,
	}

	service.Start()

	return service, srv
}

func TestGetPeerStatus(t *testing.T) {
	t.Parallel()

	clientSrv := newTestNetwork(t)
	client := newTestSyncPeerClient(clientSrv, nil)

	t.Cleanup(func() {
		client.Close()
	})

	peerLatest := uint64(10)
	_, peerSrv := createTestSyncerService(t, &mockBlockchain{
		headerHandler: newSimpleHeaderHandler(peerLatest),
	})

	err := network.JoinAndWait(
		t,
		clientSrv,
		peerSrv,
		network.DefaultBufferTimeout,
		network.DefaultJoinTimeout,
		false,
	)

	assert.NoError(t, err)

	status, err := client.GetPeerStatus(peerSrv.AddrInfo().ID)
	assert.NoError(t, err)

	expected := &NoForkPeer{
		ID:     peerSrv.AddrInfo().ID,
		Number: peerLatest,
	}

	assert.Equal(t, expected, status)
}

func TestGetConnectedPeerStatuses(t *testing.T) {
	t.Parallel()

	clientSrv := newTestNetwork(t)
	client := newTestSyncPeerClient(clientSrv, nil)

	t.Cleanup(func() {
		client.Close()
	})

	var (
		peerLatests = []uint64{
			30,
			20,
			10,
		}

		mutex        = sync.Mutex{}
		peerJoinErrs = make([]error, len(peerLatests))
		expected     = make([]*NoForkPeer, len(peerLatests))

		wg sync.WaitGroup
	)

	for idx, latest := range peerLatests {
		idx, latest := idx, latest

		_, peerSrv := createTestSyncerService(t, &mockBlockchain{
			headerHandler: newSimpleHeaderHandler(latest),
		})

		peerID := peerSrv.AddrInfo().ID

		wg.Add(1)

		go func() {
			defer wg.Done()

			mutex.Lock()
			defer mutex.Unlock()

			peerJoinErrs[idx] = network.JoinAndWait(
				t,
				clientSrv,
				peerSrv,
				network.DefaultBufferTimeout,
				network.DefaultJoinTimeout,
				false,
			)

			expected[idx] = &NoForkPeer{
				ID:     peerID,
				Number: latest,
			}
		}()
	}

	wg.Wait()

	for _, err := range peerJoinErrs {
		assert.NoError(t, err)
	}

	statuses := client.GetConnectedPeerStatuses()

	// no need to check order
	assert.Equal(t, expected, sortNoForkPeers(statuses))
}

func TestStatusPubSub(t *testing.T) {
	t.Parallel()

	clientSrv := newTestNetwork(t)
	client := newTestSyncPeerClient(clientSrv, nil)

	t.Cleanup(func() {
		client.Close()
	})

	_, peerSrv := createTestSyncerService(t, &mockBlockchain{})
	peerID := peerSrv.AddrInfo().ID

	client.subscribeEventProcess()

	// run goroutine to collect events
	var (
		events = []*event.PeerEvent{}
		wg     sync.WaitGroup
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		for event := range client.GetPeerConnectionUpdateEventCh() {
			events = append(events, event)
		}
	}()

	// connect
	err := network.JoinAndWait(
		t,
		clientSrv,
		peerSrv,
		network.DefaultBufferTimeout,
		network.DefaultJoinTimeout,
		false,
	)

	assert.NoError(t, err)

	// disconnect
	err = network.DisconnectAndWait(
		clientSrv,
		peerID,
		network.DefaultLeaveTimeout,
	)

	assert.NoError(t, err)

	// close channel and wait for events
	close(client.peerConnectionUpdateCh)

	wg.Wait()

	expected := []*event.PeerEvent{
		{
			PeerID: peerID,
			Type:   event.PeerConnected,
		},
		{
			PeerID: peerID,
			Type:   event.PeerDisconnected,
		},
	}

	assert.Equal(t, expected, events)
}

func TestPeerConnectionUpdateEventCh(t *testing.T) {
	var (
		// network layer
		clientSrv = newTestNetwork(t)
		peerSrv1  = newTestNetwork(t)
		peerSrv2  = newTestNetwork(t)
		peerSrv3  = newTestNetwork(t) // to wait for gossipped message

		// latest block height
		peerLatest1 = uint64(10)
		peerLatest2 = uint64(20)

		// blockchain subscription
		subscription1 = blockchain.NewMockSubscription()
		subscription2 = blockchain.NewMockSubscription()

		// syncer client
		client = newTestSyncPeerClient(clientSrv, &mockBlockchain{
			subscription: &blockchain.MockSubscription{},
		})
		peerClient1 = newTestSyncPeerClient(peerSrv1, &mockBlockchain{
			subscription:  subscription1,
			headerHandler: newSimpleHeaderHandler(peerLatest1),
		})
		peerClient2 = newTestSyncPeerClient(peerSrv2, &mockBlockchain{
			subscription:  subscription2,
			headerHandler: newSimpleHeaderHandler(peerLatest2),
		})
	)

	t.Cleanup(func() {
		clientSrv.Close()
		peerSrv1.Close()
		peerSrv2.Close()
		peerSrv3.Close()

		// no need to call Close of Client because test closes it manually

		client.Close()
		peerClient1.Close()
		peerClient2.Close()
	})

	// client <-> peer1
	// peer1  <-> peer2
	// peer2  <-> peer3
	err := network.JoinAndWaitMultiple(
		t,
		network.DefaultJoinTimeout,
		// client <-> peer1
		clientSrv,
		peerSrv1,
		// peer1 <-> peer2
		peerSrv1,
		peerSrv2,
		// peer2 <-> peer3
		peerSrv2,
		peerSrv3,
	)

	assert.NoError(t, err)

	// start gossip
	assert.NoError(t, client.startGossip())
	assert.NoError(t, peerClient1.startGossip())
	assert.NoError(t, peerClient2.startGossip())

	// create topic
	topic, err := peerSrv3.NewTopic(statusTopicName, &proto.SyncPeerStatus{})
	assert.NoError(t, err)

	var wgForGossip sync.WaitGroup

	// 2 messages should be gossipped
	wgForGossip.Add(2)

	handler := func(_ interface{}, _ peer.ID) {
		wgForGossip.Done()
	}

	assert.NoError(t, topic.Subscribe(handler))

	// need to wait for a few seconds to propagate subscribing
	time.Sleep(2 * time.Second)

	// enable peers to send own status via gossip
	peerClient1.EnablePublishingPeerStatus()
	peerClient2.EnablePublishingPeerStatus()

	// start to subscribe blockchain events
	go peerClient1.startNewBlockProcess()
	go peerClient2.startNewBlockProcess()

	// collect peer status changes
	var (
		wgForConnectingStatus sync.WaitGroup
		newStatuses           []*NoForkPeer
	)

	wgForConnectingStatus.Add(1)

	go func() {
		defer wgForConnectingStatus.Done()
		wgForGossip.Wait()

		for status := range client.GetPeerStatusUpdateCh() {
			newStatuses = append(newStatuses, status)

			if len(newStatuses) > 0 {
				break
			}
		}
	}()

	// push latest block number to blockchain subscription
	pushSubscription := func(sub *blockchain.MockSubscription, latest uint64) {
		sub.Push(&blockchain.Event{
			NewChain: []*types.Header{
				{
					Number: latest,
				},
			},
		})
	}

	// peer1 and peer2 emit Blockchain event
	// they should publish their status via gossip
	pushSubscription(subscription1, peerLatest1)
	pushSubscription(subscription2, peerLatest2)

	// wait until 2 messages are propagated
	wgForGossip.Wait()

	// wait until collecting routine is done
	wgForConnectingStatus.Wait()

	// client connects to only peer1, then expects to have a status from peer1
	expected := []*NoForkPeer{
		{
			ID:     peerSrv1.AddrInfo().ID,
			Number: peerLatest1,
		},
	}

	assert.Equal(t, expected, newStatuses)
}

// Make sure the peer shouldn't emit status if the shouldEmitBlocks flag is set.
// The subtests cannot contain t.Parallel() due to how
// the test is organized
//
//nolint:tparallel
func Test_shouldEmitBlocks(t *testing.T) {
	t.Parallel()

	var (
		// network layer
		clientSrv = newTestNetwork(t)
		peerSrv   = newTestNetwork(t)

		clientLatest = uint64(10)

		subscription = blockchain.NewMockSubscription()

		client = newTestSyncPeerClient(clientSrv, &mockBlockchain{
			subscription:  subscription,
			headerHandler: newSimpleHeaderHandler(clientLatest),
		})
	)

	t.Cleanup(func() {
		clientSrv.Close()
		peerSrv.Close()
		client.Close()
	})

	err := network.JoinAndWaitMultiple(
		t,
		network.DefaultJoinTimeout,
		clientSrv,
		peerSrv,
	)

	assert.NoError(t, err)

	// start gossip
	assert.NoError(t, client.startGossip())

	// start to subscribe blockchain events
	go client.startNewBlockProcess()

	// push latest block number to blockchain subscription
	pushSubscription := func(sub *blockchain.MockSubscription, latest uint64) {
		sub.Push(&blockchain.Event{
			NewChain: []*types.Header{
				{
					Number: latest,
				},
			},
		})
	}

	waitForContext := func(ctx context.Context, timer *time.Timer) bool {
		select {
		case <-ctx.Done():
			return true
		case <-timer.C:
			return false
		}
	}

	// create topic & subscribe in peer
	topic, err := peerSrv.NewTopic(statusTopicName, &proto.SyncPeerStatus{})
	assert.NoError(t, err)

	testGossip := func(t *testing.T, shouldEmit bool) {
		t.Helper()

		// context to be canceled when receiving status
		receiveContext, cancelContext := context.WithCancel(context.Background())
		defer cancelContext()

		assert.NoError(t, topic.Subscribe(func(_ interface{}, id peer.ID) {
			cancelContext()
		}))

		// need to wait for a few seconds to propagate subscribing
		time.Sleep(2 * time.Second)

		if shouldEmit {
			client.EnablePublishingPeerStatus()
		} else {
			client.DisablePublishingPeerStatus()
		}

		pushSubscription(subscription, clientLatest)

		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()

		canceled := waitForContext(receiveContext, timer)

		assert.Equal(t, shouldEmit, canceled)
	}

	t.Run("should send own status via gossip if shouldEmitBlocks is set", func(t *testing.T) {
		testGossip(t, true)
	})

	t.Run("shouldn't send own status via gossip if shouldEmitBlocks is reset", func(t *testing.T) {
		testGossip(t, false)
	})
}

func Test_syncPeerClient_GetBlocks(t *testing.T) {
	t.Parallel()

	clientSrv := newTestNetwork(t)
	client := newTestSyncPeerClient(clientSrv, nil)

	t.Cleanup(func() {
		client.Close()
	})

	var (
		peerLatest = uint64(10)
		syncFrom   = uint64(1)
	)

	_, peerSrv := createTestSyncerService(t, &mockBlockchain{
		headerHandler: newSimpleHeaderHandler(peerLatest),
		getBlockByNumberHandler: func(u uint64, b bool) (*types.Block, bool) {
			if u <= 10 {
				return &types.Block{
					Header: &types.Header{
						Number: u,
					},
				}, true
			}

			return nil, false
		},
	})

	err := network.JoinAndWait(
		t,
		clientSrv,
		peerSrv,
		network.DefaultBufferTimeout,
		network.DefaultJoinTimeout,
		false,
	)

	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	syncedBlocks, err := client.GetBlocks(ctx, peerSrv.AddrInfo().ID, syncFrom, peerLatest)
	assert.NoError(t, err)

	// hash is calculated on unmarshaling
	expected := createMockBlocks(10)
	for _, b := range expected {
		b.Header.ComputeHash()
	}

	assert.Equal(t, expected, syncedBlocks)
}

// setupIncompatibleGRPCServer setups an incompatible protocol GRPC server
func (s *syncPeerService) setupIncompatibleGRPCServer() {
	s.stream = grpc.NewGrpcStream(context.TODO())

	proto.RegisterV1Server(s.stream.GrpcServer(), s)
	s.stream.Serve()
	s.network.RegisterProtocol("/fake-syncer/1.0", s.stream)
}

func createNonSyncerService(t *testing.T, chain Blockchain) (*syncPeerService, network.Server) {
	t.Helper()

	srv := newTestNetwork(t)

	service := &syncPeerService{
		blockchain: chain,
		network:    srv,
	}

	service.setupIncompatibleGRPCServer()

	return service, srv
}

func Test_newSyncPeerClient_forgetNonProtocolPeer(t *testing.T) {
	t.Parallel()

	clientSrv := newTestNetwork(t)
	client := newTestSyncPeerClient(clientSrv, nil)

	_, peerSrv := createNonSyncerService(t, &mockBlockchain{
		headerHandler: newSimpleHeaderHandler(10),
	})
	srvID := peerSrv.AddrInfo().ID

	t.Cleanup(func() {
		client.Close()
		peerSrv.Close()
	})

	err := network.JoinAndWait(
		t,
		clientSrv,
		peerSrv,
		network.DefaultBufferTimeout,
		network.DefaultJoinTimeout,
		false,
	)

	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.GetPeerStatus(srvID)
	assert.Error(t, err)

	_, err = network.WaitUntilPeerDisconnectsFrom(ctx, clientSrv, srvID)
	assert.NoError(t, err)

	// client should disconnect with peer do not support syncer protocol
	assert.False(t, client.network.HasPeer(srvID))

	// client should be forget
	peers := client.network.Peers()
	assert.Equal(t, 0, len(peers))
}
