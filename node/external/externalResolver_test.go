package external_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/ElrondNetwork/elrond-go-sandbox/consensus"
	"github.com/ElrondNetwork/elrond-go-sandbox/data"
	"github.com/ElrondNetwork/elrond-go-sandbox/data/block"
	"github.com/ElrondNetwork/elrond-go-sandbox/dataRetriever"
	"github.com/ElrondNetwork/elrond-go-sandbox/node/external"
	"github.com/ElrondNetwork/elrond-go-sandbox/node/mock"
	"github.com/ElrondNetwork/elrond-go-sandbox/sharding"
	"github.com/stretchr/testify/assert"
)

var pk1 = []byte("pk1")
var defaultShardHeader = block.Header{
	Nonce:     1,
	ShardId:   2,
	TxCount:   3,
	TimeStamp: 4,
}
var testMarshalizer = &mock.MarshalizerFake{}

func createMockValidatorGroupSelector() consensus.ValidatorGroupSelector {
	return &mock.ValidatorGroupSelectorStub{
		ComputeValidatorsGroupCalled: func(randomness []byte) (validatorsGroup []consensus.Validator, err error) {
			return []consensus.Validator{
				&mock.ValidatorMock{PubKeyField: pk1},
			}, nil
		},
	}
}

func createMockStorer() dataRetriever.StorageService {
	return &mock.ChainStorerMock{
		GetCalled: func(unitType dataRetriever.UnitType, key []byte) (bytes []byte, e error) {
			if unitType == dataRetriever.BlockHeaderUnit {
				hdrBuff, _ := testMarshalizer.Marshal(&defaultShardHeader)
				return hdrBuff, nil
			}

			//metablocks
			//key is something like "hash_0", "hash_1"
			//so we generate a metablock that has prevHash = "hash_"+[nonce - 1]
			nonce, _ := strconv.Atoi(strings.Split(string(key), "_")[1])

			metablock := &block.MetaBlock{
				Nonce:    uint64(nonce),
				PrevHash: []byte(fmt.Sprintf("hash_%d", nonce-1)),
				//one shard info (1 metablock contains one shard header hash)
				ShardInfo: []block.ShardData{{}},
			}

			metaHdrBuff, _ := testMarshalizer.Marshal(metablock)
			return metaHdrBuff, nil
		},
	}
}

//------- NewExternalResolver

func TestNewExternalResolver_NilCoordinatorShouldErr(t *testing.T) {
	t.Parallel()

	ner, err := external.NewExternalResolver(
		nil,
		&mock.BlockChainMock{},
		&mock.ChainStorerMock{},
		&mock.MarshalizerMock{},
		&mock.ValidatorGroupSelectorStub{},
	)

	assert.Nil(t, ner)
	assert.Equal(t, external.ErrNilShardCoordinator, err)
}

func TestNewExternalResolver_NilChainHandlerShouldErr(t *testing.T) {
	t.Parallel()

	ner, err := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{},
		nil,
		&mock.ChainStorerMock{},
		&mock.MarshalizerMock{},
		&mock.ValidatorGroupSelectorStub{},
	)

	assert.Nil(t, ner)
	assert.Equal(t, external.ErrNilBlockChain, err)
}

func TestNewExternalResolver_NilChainStorerShouldErr(t *testing.T) {
	t.Parallel()

	ner, err := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{},
		&mock.BlockChainMock{},
		nil,
		&mock.MarshalizerMock{},
		&mock.ValidatorGroupSelectorStub{},
	)

	assert.Nil(t, ner)
	assert.Equal(t, external.ErrNilStore, err)
}

func TestNewExternalResolver_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	ner, err := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{},
		&mock.BlockChainMock{},
		&mock.ChainStorerMock{},
		nil,
		&mock.ValidatorGroupSelectorStub{},
	)

	assert.Nil(t, ner)
	assert.Equal(t, external.ErrNilMarshalizer, err)
}

func TestNewExternalResolver_NilValidatorGroupSelectorShouldErr(t *testing.T) {
	t.Parallel()

	ner, err := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{},
		&mock.BlockChainMock{},
		&mock.ChainStorerMock{},
		&mock.MarshalizerMock{},
		nil,
	)

	assert.Nil(t, ner)
	assert.Equal(t, external.ErrNilValidatorGroupSelector, err)
}

func TestNewExternalResolver_OkValsShouldWork(t *testing.T) {
	t.Parallel()

	ner, err := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{},
		&mock.BlockChainMock{},
		&mock.ChainStorerMock{},
		&mock.MarshalizerMock{},
		&mock.ValidatorGroupSelectorStub{},
	)

	assert.NotNil(t, ner)
	assert.Nil(t, err)
}

//------ RecentNotarizedBlocks

func TestExternalResolver_RecentNotarizedBlocksNotMetachainShouldErr(t *testing.T) {
	t.Parallel()

	ner, _ := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{},
		&mock.BlockChainMock{},
		&mock.ChainStorerMock{},
		&mock.MarshalizerMock{},
		&mock.ValidatorGroupSelectorStub{},
	)

	recentBlocks, err := ner.RecentNotarizedBlocks(1)
	assert.Nil(t, recentBlocks)
	assert.Equal(t, external.ErrOperationNotSupported, err)
}

func TestExternalResolver_InvalidMaxNumShouldErr(t *testing.T) {
	t.Parallel()

	ner, _ := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{
			SelfIdField: sharding.MetachainShardId,
		},
		&mock.BlockChainMock{},
		&mock.ChainStorerMock{},
		&mock.MarshalizerMock{},
		&mock.ValidatorGroupSelectorStub{},
	)

	recentBlocks, err := ner.RecentNotarizedBlocks(0)
	assert.Nil(t, recentBlocks)
	assert.Equal(t, external.ErrInvalidValue, err)
}

func TestExternalResolver_CurrentBlockIsGenesisShouldWork(t *testing.T) {
	t.Parallel()

	ner, _ := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{
			SelfIdField: sharding.MetachainShardId,
		},
		&mock.BlockChainMock{
			GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
				return &block.MetaBlock{
					Nonce: 1,
				}
			},
		},
		&mock.ChainStorerMock{},
		&mock.MarshalizerMock{},
		&mock.ValidatorGroupSelectorStub{},
	)

	recentBlocks, err := ner.RecentNotarizedBlocks(1)
	assert.Empty(t, recentBlocks)
	assert.Nil(t, err)
}

func TestExternalResolver_WithBlocksShouldWork(t *testing.T) {
	t.Parallel()

	crtNonce := 5

	ner, _ := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{
			SelfIdField: sharding.MetachainShardId,
		},
		&mock.BlockChainMock{
			GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
				return &block.MetaBlock{
					Nonce:     uint64(crtNonce),
					PrevHash:  []byte(fmt.Sprintf("hash_%d", crtNonce-1)),
					ShardInfo: []block.ShardData{{}},
				}
			},
		},
		createMockStorer(),
		testMarshalizer,
		createMockValidatorGroupSelector(),
	)

	recentBlocks, err := ner.RecentNotarizedBlocks(crtNonce)
	//should have received 4 shard blocks since meta block with nonce 0 is considered empty
	assert.Equal(t, crtNonce-1, len(recentBlocks))
	assert.Nil(t, err)
}

func TestExternalResolver_WithMoreBlocksShouldWorkAndReturnMaxBlocksNum(t *testing.T) {
	t.Parallel()

	crtNonce := 50

	ner, _ := external.NewExternalResolver(
		&mock.ShardCoordinatorMock{
			SelfIdField: sharding.MetachainShardId,
		},
		&mock.BlockChainMock{
			GetCurrentBlockHeaderCalled: func() data.HeaderHandler {
				return &block.MetaBlock{
					Nonce:     uint64(crtNonce),
					PrevHash:  []byte(fmt.Sprintf("hash_%d", crtNonce-1)),
					ShardInfo: []block.ShardData{{}},
				}
			},
		},
		createMockStorer(),
		testMarshalizer,
		createMockValidatorGroupSelector(),
	)

	maxBlocks := 10

	recentBlocks, err := ner.RecentNotarizedBlocks(maxBlocks)
	assert.Equal(t, maxBlocks, len(recentBlocks))
	assert.Nil(t, err)
}
