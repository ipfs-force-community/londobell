package model

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/batch"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin/v18/verifreg"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/ipfs-force-community/londobell/common"
	"github.com/ipfs-force-community/londobell/lib/mgoutil/mcodec"
	"github.com/ipfs-force-community/londobell/lib/mir"
	"github.com/ipfs-force-community/londobell/racailum/segment/extract"

	cbg "github.com/whyrusleeping/cbor-gen"
)

func init() {
	mcodec.RegisterSchemaType(new(ExecTraceReturn))
}

// NSGasTraceNames is the name of the dict GasTraceNames
const NSGasTraceNames = "GasTraceNames"

var (
	execTraceColName    = getColName(ExecTrace{})
	execTraceEpochField = extractEpochFieldName(ExecTrace{})
	execGasColName      = getColName(ExecGas{})
	execGasEpochField   = extractEpochFieldName(ExecGas{})
)

var (
	_ common.IndexedDocument = (*ExecTrace)(nil)
	_ common.Document        = (*ExecGas)(nil)
)

// NewExecTrace converts raw exec trace struct to ExecTrace*
func NewExecTrace(
	ctx *extract.Ctx,
	mcid cid.Cid,
	signedCid cid.Cid,
	epoch abi.ChainEpoch,
	seq []int,
	raw *common.ExecutionTraceCompact,
	returnObj cbor.Er,
	cost *api.MsgGasCost, meth string, isBlock bool, IDCidMap map[string][2]cid.Cid,
	rootMsgRct *types.MessageReceipt,
) (*ExecTrace, *ExecGas, error) {
	me := &ExecTrace{
		Cid:          mcid,
		SignedCid:    signedCid,
		Epoch:        epoch,
		Seq:          seq,
		Depth:        len(seq),
		Ver:          build.CurrentCommit,
		SubCallCount: len(raw.Subcalls),
		GasCost:      cost,
		IsBlock:      isBlock,
	}
	elog := ctx.L.With("NewExecTrace", mcid)
	if err := mir.Mirror(me, raw); err != nil {
		return nil, nil, fmt.Errorf("mirroring message exec: %w", err)
	}

	me.Msg.MethodName = meth
	me.SeqIndex = make([][]int, len(seq))
	me.FIL = CalculateFILValue(me.Msg.Value.String())
	for i := range seq {
		me.SeqIndex[i] = seq[:i+1]
	}

	if len(raw.MsgRct.Return) > 0 && returnObj != nil {
		if err := returnObj.UnmarshalCBOR(bytes.NewReader(raw.MsgRct.Return)); err != nil {
			// 兼容 verifreg.ClaimAllocations 返回值的新格式
			// 旧版 go-state-types 定义: { BatchInfo BatchReturn, ClaimedSpace big.Int }
			// 链上实际返回(v18+): { sector_results BatchReturn, sector_claims []SectorClaimSummary }
			if meth == "ClaimAllocations" {
				if compatRet, compatErr := unmarshalClaimAllocationsReturnV18(raw.MsgRct.Return); compatErr == nil {
					// 将新格式转换为旧格式,便于下游消费
					oldRet := &verifreg.ClaimAllocationsReturn{
						BatchInfo:    compatRet.SectorResults,
						ClaimedSpace: compatRet.TotalClaimedSpace(),
					}
					me.Detail.Return = oldRet
				} else {
					return nil, nil, fmt.Errorf("unmarshal return: %w (also failed to decode new ClaimAllocations format: %v)", err, compatErr)
				}
			} else {
				return nil, nil, fmt.Errorf("unmarshal return: %w", err)
			}
		} else {
			me.Detail.Return = returnObj
		}
	}

	me.genID()
	err := me.genRootids(IDCidMap)
	if err != nil {
		elog.Warn(err)
	}
	if me.Depth == 1 {
		me.MsgRct.EventsRoot = rootMsgRct.EventsRoot
		me.MsgRct.GasUsed = rootMsgRct.GasUsed
	}
	//var mg *ExecGas
	//
	//if len(raw.GasCharges) > 0 {
	//	mg = &ExecGas{
	//		Epoch:   epoch,
	//		Charges: make([]common.GasTraceCompact, len(raw.GasCharges)),
	//	}
	//
	//	for i := range raw.GasCharges {
	//		charge := raw.GasCharges[i]
	//
	//		nameIdx, err := dal.LookupEnum(ctx, NSGasTraceNames, charge.Name)
	//		if err != nil {
	//			return nil, nil, fmt.Errorf("lookup for gas-trace-name index for %s in dict: %w", charge.Name, err)
	//		}
	//
	//		charge.Name = fmt.Sprintf("$%d", nameIdx)
	//		mg.Charges[i] = charge
	//	}
	//
	//	mg.ID = me.ID
	//}

	return me, nil, nil
}

// ExecTraceReturn is a type alias
type ExecTraceReturn cbor.Er

// ExecTrace is the schema of *api.ExecutionTrace
type ExecTrace struct {
	ID string `mir:"-" bson:"_id"`

	Cid       cid.Cid        `mir:"-"`
	SignedCid cid.Cid        `mir:"-" bson:"SignedCid,omitempty"`
	Epoch     abi.ChainEpoch `mir:"-"`
	Seq       []int          `mir:"-"`
	Depth     int            `mir:"-"`
	FIL       int64
	Ver       string `mir:"-"`

	Msg struct {
		From       address.Address
		To         address.Address
		Method     abi.MethodNum
		Value      abi.TokenAmount
		MethodName string
	}

	// raw infos
	MsgRct struct {
		ExitCode    exitcode.ExitCode
		Return      []byte
		ReturnCodec uint64
		EventsRoot  *cid.Cid `mir:"-"`
		GasUsed     int64    `mir:"-"`
	}
	Error string

	SeqIndex [][]int `mir:"-"`

	SubCallCount int `mir:"-"`

	Detail struct {
		Return ExecTraceReturn
	} `mir:"-"`

	GasCost       *api.MsgGasCost `mir:"-"`
	RootCid       cid.Cid         `mir:"-"`
	RootSignedCid cid.Cid         `mir:"-"`
	IsBlock       bool
}

// Indexes impl common.Indexed
func (et *ExecTrace) Indexes() [][]string {
	return [][]string{
		[]string{execTraceEpochField, "Msg.To", "Msg.Method", "MsgRct.ExitCode"},
		[]string{execTraceEpochField, "Msg.To", "Seq"},
		[]string{"Cid"},
		[]string{"SignedCid"},
		[]string{"Depth", execTraceEpochField},
		[]string{"Depth", "Msg.MethodName", execTraceEpochField},
		[]string{"FIL"},
	}
}

// CollectionName impls common.Document
func (et *ExecTrace) CollectionName() string {
	return execTraceColName
}

// EpochField impl common.Document
func (et *ExecTrace) EpochField() *string {
	return &execTraceEpochField
}

// ResetPolicy impls common.Document
func (et *ExecTrace) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(execTraceEpochField, lower, upper), true
}

func (et *ExecTrace) IsMutable() bool {
	return false
}

func (et *ExecTrace) genID() {
	seqStrs := make([]string, 0, len(et.Seq))
	for i := range et.Seq {
		seqStrs = append(seqStrs, fmt.Sprintf("%05d", et.Seq[i]))
	}

	et.ID = fmt.Sprintf("%d-%s", et.Epoch, strings.Join(seqStrs, "-"))
}

// get root Cid SignedCid
func (et *ExecTrace) genRootids(m map[string][2]cid.Cid) error {
	if et.IsBlock {
		return nil
	}
	rootID, err := GetRootID(et.ID)
	if err != nil {
		return err
	}
	et.RootCid = m[rootID][0]
	et.RootSignedCid = m[rootID][1]
	return nil
}

// ExecGas stores gas charges in another collection
type ExecGas struct {
	ID      string `mir:"-" bson:"_id"`
	Epoch   abi.ChainEpoch
	Charges []common.GasTraceCompact
}

// CollectionName impls common.Document
func (eg *ExecGas) CollectionName() string {
	return execGasColName
}

// EpochField impl common.Document
func (eg *ExecGas) EpochField() *string {
	return &execGasEpochField
}

// ResetPolicy impls common.Document
func (eg *ExecGas) ResetPolicy(lower, upper *abi.ChainEpoch) (interface{}, bool) {
	return rangedFilter(execGasEpochField, lower, upper), true
}

func (eg *ExecGas) IsMutable() bool {
	return false
}

// 计算 "FIL" 字段的值
func CalculateFILValue(value string) int64 {
	// 如果 "value" 字段长度大于18，截取前 len-18 个字符并转化为 int64
	if len(value) > 18 {
		prefix := value[:len(value)-18]
		FILValue, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return 0 // 处理错误情况
		}
		return FILValue
	}

	return 0 // 如果长度不足18，返回0
}

// sectorClaimSummary 对应 builtin-actors 中的 SectorClaimSummary
// #[serde(transparent)] 表示它就是一个 big.Int
type sectorClaimSummary struct {
	ClaimedSpace big.Int
}

func (t *sectorClaimSummary) MarshalCBOR(w io.Writer) error {
	return t.ClaimedSpace.MarshalCBOR(w)
}

func (t *sectorClaimSummary) UnmarshalCBOR(r io.Reader) error {
	return t.ClaimedSpace.UnmarshalCBOR(r)
}

// claimAllocationsReturnV18 对应链上 v18+ 的新格式 ClaimAllocationsReturn
// struct ClaimAllocationsReturn { sector_results: BatchReturn, sector_claims: Vec<SectorClaimSummary> }
type claimAllocationsReturnV18 struct {
	SectorResults batch.BatchReturn
	SectorClaims  []sectorClaimSummary
}

// TotalClaimedSpace 汇总所有 sector 的 claimed_space
func (t *claimAllocationsReturnV18) TotalClaimedSpace() big.Int {
	total := big.Zero()
	for i := range t.SectorClaims {
		total = big.Add(total, t.SectorClaims[i].ClaimedSpace)
	}
	return total
}

// unmarshalClaimAllocationsReturnV18 尝试用新格式解码 ClaimAllocations 返回值
func unmarshalClaimAllocationsReturnV18(data []byte) (*claimAllocationsReturnV18, error) {
	t := &claimAllocationsReturnV18{}
	cr := cbg.NewCborReader(bytes.NewReader(data))

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajArray {
		return nil, fmt.Errorf("cbor input should be of type array")
	}
	if extra != 2 {
		return nil, fmt.Errorf("cbor input had wrong number of fields, expected 2, got %d", extra)
	}

	// sector_results (BatchReturn)
	if err := t.SectorResults.UnmarshalCBOR(cr); err != nil {
		return nil, fmt.Errorf("unmarshaling sector_results: %w", err)
	}

	// sector_claims ([]SectorClaimSummary)
	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("reading sector_claims header: %w", err)
	}
	if maj != cbg.MajArray {
		return nil, fmt.Errorf("sector_claims should be of type array, got %d", maj)
	}

	if extra > 0 {
		t.SectorClaims = make([]sectorClaimSummary, extra)
		for i := range t.SectorClaims {
			if err := t.SectorClaims[i].UnmarshalCBOR(cr); err != nil {
				return nil, fmt.Errorf("unmarshaling sector_claims[%d]: %w", i, err)
			}
		}
	}

	return t, nil
}
