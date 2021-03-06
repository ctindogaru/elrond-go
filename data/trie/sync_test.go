package trie

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ElrondNetwork/elrond-go/core/check"
	"github.com/ElrondNetwork/elrond-go/data"
	"github.com/ElrondNetwork/elrond-go/data/mock"
	"github.com/ElrondNetwork/elrond-go/data/trie/statistics"
	"github.com/ElrondNetwork/elrond-go/testscommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockArgument() ArgTrieSyncer {
	return ArgTrieSyncer{
		RequestHandler:                 &mock.RequestHandlerStub{},
		InterceptedNodes:               testscommon.NewCacherMock(),
		Trie:                           &patriciaMerkleTrie{},
		ShardId:                        0,
		Topic:                          "topic",
		TrieSyncStatistics:             statistics.NewTrieSyncStatistics(),
		TimeoutBetweenTrieNodesCommits: minTimeoutBetweenNodesCommits,
		MaxHardCapForMissingNodes:      1,
	}
}

func TestNewTrieSyncer_NilRequestHandlerShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.RequestHandler = nil

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.Equal(t, err, ErrNilRequestHandler)
}

func TestNewTrieSyncer_NilInterceptedNodesShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.InterceptedNodes = nil

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.Equal(t, err, data.ErrNilCacher)
}

func TestNewTrieSyncer_NilTrieShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.Trie = nil

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.Equal(t, err, ErrNilTrie)
}

func TestNewTrieSyncer_EmptyTopicShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.Topic = ""

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.Equal(t, err, ErrInvalidTrieTopic)
}

func TestNewTrieSyncer_NilTrieStatisticsShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.TrieSyncStatistics = nil

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.Equal(t, err, ErrNilTrieSyncStatistics)
}

func TestNewTrieSyncer_InvalidTimeoutBetweenTrieNodesCommitsShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.TimeoutBetweenTrieNodesCommits = time.Duration(0)

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.True(t, errors.Is(err, ErrInvalidTimeout))
}

func TestNewTrieSyncer_InvalidMaxHardCapForMissingNodesShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.MaxHardCapForMissingNodes = 0

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.True(t, errors.Is(err, ErrInvalidMaxHardCapForMissingNodes))
}

func TestNewTrieSyncer_NotACorrectTrieShouldErr(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()
	arg.Trie = &mock.TrieStub{}

	ts, err := NewTrieSyncer(arg)
	assert.True(t, check.IfNil(ts))
	assert.True(t, errors.Is(err, ErrWrongTypeAssertion))
}

func TestNewTrieSyncer_ShouldWork(t *testing.T) {
	t.Parallel()

	arg := createMockArgument()

	ts, err := NewTrieSyncer(arg)
	assert.False(t, check.IfNil(ts))
	assert.Nil(t, err)
}

func TestTrieSync_InterceptedNodeShouldNotBeAddedToNodesForTrieIfNodeReceived(t *testing.T) {
	t.Parallel()

	marshalizer, hasher := getTestMarshalizerAndHasher()
	arg := ArgTrieSyncer{
		RequestHandler:                 &mock.RequestHandlerStub{},
		InterceptedNodes:               testscommon.NewCacherMock(),
		Trie:                           &patriciaMerkleTrie{},
		ShardId:                        0,
		Topic:                          "trieNodes",
		TrieSyncStatistics:             statistics.NewTrieSyncStatistics(),
		TimeoutBetweenTrieNodesCommits: time.Second * 10,
		MaxHardCapForMissingNodes:      500,
	}
	ts, err := NewTrieSyncer(arg)
	require.Nil(t, err)

	bn, collapsedBn := getBnAndCollapsedBn(marshalizer, hasher)
	encodedNode, err := collapsedBn.getEncodedNode()
	assert.Nil(t, err)

	interceptedNode, err := NewInterceptedTrieNode(encodedNode, marshalizer, hasher)
	assert.Nil(t, err)

	hash := "nodeHash"
	ts.nodesForTrie[hash] = trieNodeInfo{
		trieNode: bn,
		received: true,
	}

	ts.trieNodeIntercepted([]byte(hash), interceptedNode)

	nodeInfo, ok := ts.nodesForTrie[hash]
	assert.True(t, ok)
	assert.Equal(t, bn, nodeInfo.trieNode)
}

func TestTrieSync_InterceptedNodeTimedOut(t *testing.T) {
	t.Parallel()

	timeout := time.Second * 2
	arg := ArgTrieSyncer{
		RequestHandler:   &mock.RequestHandlerStub{},
		InterceptedNodes: testscommon.NewCacherMock(),
		Trie: &patriciaMerkleTrie{
			trieStorage: &mock.StorageManagerStub{
				DatabaseCalled: func() data.DBWriteCacher {
					return mock.NewMemDbMock()
				},
			},
		},
		ShardId:                        0,
		Topic:                          "trieNodes",
		TrieSyncStatistics:             statistics.NewTrieSyncStatistics(),
		TimeoutBetweenTrieNodesCommits: timeout,
		MaxHardCapForMissingNodes:      500,
	}
	ts, err := NewTrieSyncer(arg)
	require.Nil(t, err)

	start := time.Now()
	err = ts.StartSyncing([]byte("roothash"), context.Background())
	end := time.Now()

	assert.True(t, errors.Is(err, ErrTimeIsOut))
	assert.True(t, timeout <= end.Sub(start))
}

func TestTrieSync_FoundInStorageShouldNotRequest(t *testing.T) {
	t.Parallel()

	timeout := time.Second * 200
	marshalizer, hasher := getTestMarshalizerAndHasher()
	bn, _ := getBnAndCollapsedBn(marshalizer, hasher)
	err := bn.setHash()
	require.Nil(t, err)
	rootHash := bn.getHash()
	db := mock.NewMemDbMock()

	err = bn.commit(true, 2, 2, db, db)
	require.Nil(t, err)

	arg := ArgTrieSyncer{
		RequestHandler: &mock.RequestHandlerStub{
			RequestTrieNodesCalled: func(destShardID uint32, hashes [][]byte, topic string) {
				assert.Fail(t, "should have not requested trie nodes")
			},
		},
		InterceptedNodes: testscommon.NewCacherMock(),
		Trie: &patriciaMerkleTrie{
			trieStorage: &mock.StorageManagerStub{
				DatabaseCalled: func() data.DBWriteCacher {
					return db
				},
			},
			marshalizer: marshalizer,
			hasher:      hasher,
		},
		ShardId:                        0,
		Topic:                          "trieNodes",
		TrieSyncStatistics:             statistics.NewTrieSyncStatistics(),
		TimeoutBetweenTrieNodesCommits: timeout,
		MaxHardCapForMissingNodes:      500,
	}
	ts, err := NewTrieSyncer(arg)
	require.Nil(t, err)

	err = ts.StartSyncing(rootHash, context.Background())
	assert.Nil(t, err)
}
