package common

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/network"
	bstore "github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/lotus/chain/state"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// HeadNotifier receives head change events from chain syncer
type HeadNotifier interface {
	Sub(ctx context.Context) (<-chan types.TipSetKey, error)
}

// DAL is the abstraction of the chain data access level
type DAL interface {
	ChainStore
	StateManager
	ChainDict
}

// ChainStore is the abstraction of chain storage
type ChainStore interface {
	LoadTipSet(ctx context.Context, tsk types.TipSetKey) (*types.TipSet, error)
	Weight(ctx context.Context, ts *types.TipSet) (types.BigInt, error)
	ActorStore(ctx context.Context) adt.Store
	GetGenesis(ctx context.Context) (*types.BlockHeader, error)
	ChainBlockstore() bstore.Blockstore
	MessagesForBlock(ctx context.Context, b *types.BlockHeader) ([]*types.Message, []*types.SignedMessage, error)
	ComputeBaseFee(ctx context.Context, ts *types.TipSet) (abi.TokenAmount, error)
	MessagesForTipset(ctx context.Context, ts *types.TipSet) ([]types.ChainMsg, error)
}

// StateManager manages the state on chain
type StateManager interface {
	ExecutionTrace(ctx context.Context, ts *types.TipSet) (cid.Cid, []*api.InvocResult, error)
	ParentState(ts *types.TipSet) (*state.StateTree, error)
	ParentStateTsk(ctx context.Context, tsk types.TipSetKey) (*state.StateTree, error)
	StateTree(st cid.Cid) (*state.StateTree, error)
	LoadActor(ctx context.Context, addr address.Address, ts *types.TipSet) (*types.Actor, error)
	LoadActorTsk(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	LoadActorRaw(_ context.Context, addr address.Address, st cid.Cid) (*types.Actor, error)
	GetNetworkVersion(ctx context.Context, height abi.ChainEpoch) network.Version
	GetVMCirculatingSupplyDetailed(ctx context.Context, height abi.ChainEpoch, st *state.StateTree) (api.CirculatingSupply, error)
	SearchForMessage(ctx context.Context, head *types.TipSet, mcid cid.Cid, lookbackLimit abi.ChainEpoch, allowReplaced bool) (*types.TipSet, *types.MessageReceipt, cid.Cid, error)
	TipSetState(ctx context.Context, ts *types.TipSet) (st cid.Cid, rec cid.Cid, err error)
	LookupIDAddress(_ context.Context, addr address.Address, ts *types.TipSet) (address.Address, error)
	LookupID(ctx context.Context, addr address.Address, ts *types.TipSet) (abi.ActorID, error)
}

// ChainDict is a dict for enums
type ChainDict interface {
	AddEnum(ctx context.Context, ns string, entry ...string) error
	LookupEnum(ctx context.Context, ns string, entry string) (int, error)
}

// MetaManager manages all meta data items, and is able to watch the changes of the specific item
type MetaManager interface {
	Load(ctx context.Context, key string, out interface{}) (bool, error)
	Update(ctx context.Context, key string, val interface{}) error
	Watch(ctx context.Context, key string, cb func(bson.RawValue) error) error
}

// Indexed will return pre-planed indexes
type Indexed interface {
	Indexes() [][]string
}

// Document is the abstraction of a stored item in a document database
type Document interface {
	CollectionName() string
	EpochField() *string
	ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool)
	IsMutable() bool
}

// IndexedDocument will return pre-planed indexes
type IndexedDocument interface {
	Indexed
	Document
}

// DocumentDB is a simple abstraction of a document databse with only insert & delete methods exported
type DocumentDB interface {
	Insert(ctx context.Context, col string, docs []interface{}) (int, error)
	Find(ctx context.Context, col string, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error)
	Update(ctx context.Context, col string, filter, docs interface{}) (*mongo.UpdateResult, error)
	Delete(ctx context.Context, col string, filter interface{}) (int, error)
	Aggregate(ctx context.Context, col string, pipeline interface{}, res interface{}) error
	FindOneAndUpdate(ctx context.Context, col string, filter interface{},
		update interface{}) error
	CountDocuments(ctx context.Context, col string, filter interface{}) (int64, error)
	FindOne(ctx context.Context, col string, filter interface{},
		opts ...*options.FindOneOptions) *mongo.SingleResult
}

// DetailPrinter prints more detailed info
type DetailPrinter interface {
	PrintDetail(log *zap.SugaredLogger)
}

// FullNodeApiGetter will return an appropriate API
type FullNodeApiGetter interface {
	GetAppropriateAPI() v0api.FullNode
}

// aliases for variables and methods
var (
	CidBuilder = abi.CidBuilder
)

// ActorHead contains the key fields in ActorState
type ActorHead struct {
	*types.Actor
	*api.CirculatingSupply

	Global struct {
		Power *types.Actor
	}

	Addr   address.Address
	Epoch  abi.ChainEpoch
	TipSet *types.TipSet
}

// InvocResultCompact is the compact representation of api.InvocResult

type InvocResultCompact struct {
	MsgCid cid.Cid

	RawMsg *types.Message `mir:"Msg"`

	MsgRct         *types.MessageReceipt
	Error          string
	GasCost        api.MsgGasCost
	ExecutionTrace ExecutionTraceCompact
}

// ExecutionTraceCompact is the compact representation of types.ExecutionTrace

type ExecutionTraceCompact struct {
	Msg      types.MessageTrace
	MsgRct   types.ReturnTrace
	Error    string
	Duration time.Duration

	GasCharges []GasTraceCompact
	Subcalls   []ExecutionTraceCompact
}

// GasTraceCompact is the compact representation of types.GasTrace

type GasTraceCompact struct {
	Name string

	// T int64 `mir:"TotalGas"`
	C int64 `mir:"ComputeGas"`
	S int64 `mir:"StorageGas"`

	TG int64 `mir:"TotalGas"`
}

// LinkedTipSet represets a normal tipset with its next epoch
type LinkedTipSet struct {
	*types.TipSet
	Child  *types.TipSet
	Parent *types.TipSet
}

// State returns the state root of current tipset, from its next tipset
func (lts *LinkedTipSet) State() cid.Cid {
	if lts.Child == nil {
		return cid.Undef
	}

	return lts.Child.ParentState()
}

func (lts *LinkedTipSet) String() string {
	if lts == nil {
		return "nil"
	}

	return FormatTipSet(lts.TipSet)
}
