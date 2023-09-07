package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/builtin/v11/miner"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/api/v0api"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	verifregtypes "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"
	"github.com/filecoin-project/lotus/api/client"
	lminer "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/londobell/common"
)

var log = logging.Logger("extend")

func main() {
	logging.SetLogLevel("vm", "ERROR")

	app := &cli.App{
		Name:  "extend",
		Usage: "extend sector",
		Commands: []*cli.Command{
			alertCmd,
			queryCmd,
			extendSectorCmd,
			extendClaimCmd,
		},
		EnableBashCompletion: true,
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}

var alertCmd = &cli.Command{
	Name: "alert",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "miner",
			Required: true,
		},
		&cli.Int64Flag{
			Name:  "dead-duration",
			Value: 12,
		},
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
		},
		&cli.StringFlag{
			Name: "token",
		},
		&cli.DurationFlag{
			Name:  "tick",
			Usage: "--tick 1d",
		},
		&cli.StringFlag{
			Name:  "from-email",
			Usage: "发送邮箱地址",
		},
		&cli.StringFlag{
			Name:  "smtp-code",
			Usage: "发送邮箱授权码",
		},
		&cli.StringFlag{
			Name:  "to-email",
			Usage: "邮箱地址",
		},
	},
	Action: func(cctx *cli.Context) error {
		// 到期报警程序 for循环定时拉sectorinfo
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// email config
		smtpHost := "smtp.qq.com"
		smtpPort := 587
		from := cctx.String("from-email")
		password := cctx.String("smtp-code")
		to := cctx.String("to")
		auth := smtp.PlainAuth("", from, password, smtpHost)
		subject := "expiring sectors and claims alert"

		miner, err := address.NewFromString(cctx.String("miner"))
		if err != nil {
			return err
		}

		var requestHeader http.Header
		token := cctx.String("token")
		url := cctx.String("api-url")
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			return err
		}
		defer closer()

		var duration time.Duration
		if !cctx.IsSet("tick") {
			duration = 24 * time.Hour
		} else {
			duration = cctx.Duration("tick")
		}

		tick := time.NewTicker(duration)
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				sectors, err := api.StateMinerSectors(ctx, miner, nil, types.EmptyTSK)
				if err != nil {
					return err
				}

				expiringSectors := make([]*lminer.SectorOnChainInfo, 0)
				expiringClaims := make(map[verifregtypes.ClaimId]verifregtypes.Claim, 0)
				for _, sector := range sectors {
					if sector.SectorNumber == 10091 {
						fmt.Println(sector.Expiration, GetCurEpoch())
					}
					if sector.Expiration > GetCurEpoch() && sector.Expiration-GetCurEpoch() < abi.ChainEpoch(cctx.Int64("dead-duration")) {
						expiringSectors = append(expiringSectors, sector)
					}
				}

				outExpiringSectors, err := json.MarshalIndent(expiringSectors, "", "  ")
				if err != nil {
					return err
				}

				claimMap, err := api.StateGetClaims(ctx, miner, types.EmptyTSK)
				if err != nil {
					return err
				}

				for id, claim := range claimMap {
					// 已过期的也可以再续期
					if claim.TermStart+claim.TermMax > GetCurEpoch() && claim.TermStart+claim.TermMax-GetCurEpoch() < abi.ChainEpoch(cctx.Int64("dead-duration")) || claim.TermStart+claim.TermMax <= common.GetCurEpoch() {
						expiringClaims[id] = claim
					}
				}

				outExpiringClaims, err := json.MarshalIndent(expiringClaims, "", "  ")
				if err != nil {
					return err
				}

				// send email
				body := ""
				if len(expiringSectors) > 0 {
					body += fmt.Sprintf("expiring sectors: %+v", string(outExpiringSectors))
				}
				if len(expiringClaims) > 0 {
					body += fmt.Sprintf("expiring claims: %+v", string(outExpiringClaims))
				}
				message := []byte("To: " + to + "\r\n" +
					"From: " + from + "\r\n" +
					"Subject: " + subject + "\r\n" +
					"\r\n" +
					body + "\r\n")

				err = smtp.SendMail(smtpHost+":"+fmt.Sprint(smtpPort), auth, from, []string{to}, message)
				if err != nil {
					log.Fatal(err)
				}

				log.Info("Email sent successfully!")
			case <-ctx.Done():
				log.Infof("ctx done!!")
			}
		}
	},
}

var queryCmd = &cli.Command{
	Name: "query",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "miner",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
		},
		&cli.StringFlag{
			Name: "token",
		},
		&cli.Int64Flag{
			Name: "number",
		},
		&cli.StringFlag{
			Name:  "tipset",
			Usage: "tipset key(pass comma separated array of cids), default chainhead",
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: "sector or claim",
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var requestHeader http.Header
		token := cctx.String("token")
		url := cctx.String("api-url")
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			return err
		}
		defer closer()

		number := cctx.Int64("number")
		tipsetKey := cctx.String("tipset")
		ts, err := NewStringForTipSet(ctx, tipsetKey, api)
		if err != nil {
			return err
		}

		miner, err := address.NewFromString(cctx.String("miner"))
		if err != nil {
			log.Errorf("miner %v is invalid for query sector or claim: %v", cctx.String("miner"), err)
			return err
		}

		qtype := cctx.String("type")
		switch qtype {
		case "sector":
			sectorInfo, err := api.StateSectorGetInfo(ctx, miner, abi.SectorNumber(number), ts.Key())
			if err != nil {
				return err
			}

			formatSectorInfo, err := json.MarshalIndent(sectorInfo, "", "  ")
			if err != nil {
				return err
			}

			log.Infof("sectorInfo of %v for %v: %+v", miner, number, formatSectorInfo)
		case "claim":
			claim, err := api.StateGetClaim(ctx, miner, verifregtypes.ClaimId(number), ts.Key())
			if err != nil {
				return err
			}

			formatClaim, err := json.MarshalIndent(claim, "", "  ")
			if err != nil {
				return err
			}

			log.Infof("claim of %v for %v: %+v", miner, number, formatClaim)
		default:
			return fmt.Errorf("invalid query type: %v", qtype)
		}

		return nil
	},
}

// todo: 合理化地续期， 提供建议; 自动续期
// api写权限
var extendSectorCmd = &cli.Command{
	Name: "extend-sector",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
		},
		&cli.StringFlag{
			Name: "token",
		},
		&cli.StringFlag{
			Name:  "tipset",
			Usage: "tipset key(pass comma separated array of cids), default chainhead",
		},
		&cli.StringFlag{
			Name:     "from",
			Required: true,
			Usage:    "message of from",
		},
		&cli.StringFlag{
			Name:     "miner",
			Required: true,
		},
		// todo: 多个deadline
		&cli.Uint64Flag{
			Name:     "deadline",
			Required: true,
		},
		&cli.Uint64Flag{
			Name:     "partition",
			Required: true,
		},
		&cli.Uint64SliceFlag{
			Name:     "sectors",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "new-expiration",
			Required: true,
			// 建议
		},
		&cli.StringFlag{
			Name:  "sectors-with-claims",
			Usage: "json to store SectorsWithClaims, e.g.: ",
			// 建议
		},
		&cli.BoolFlag{
			Name:  "new",
			Usage: "ExtendSectorExpiration or ExtendSectorExpiration2",
			// 建议
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var requestHeader http.Header
		token := cctx.String("token")
		url := cctx.String("api-url")
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			return err
		}
		defer closer()

		sectorsWithClaimsJSON := cctx.String("sectors-with-claims")
		var (
			sectorClaims []miner.SectorClaim
		)
		if len(sectorsWithClaimsJSON) > 0 {
			sectorClaims, err = ParseSectorClaims(sectorsWithClaimsJSON)
			if err != nil {
				return err
			}
		}

		fromAddr := cctx.String("from")
		toAddr := cctx.String("to")
		tipsetKey := cctx.String("tipset")

		ts, err := NewStringForTipSet(ctx, tipsetKey, api)
		if err != nil {
			return err
		}

		from, err := address.NewFromString(fromAddr)
		if err != nil {
			return err
		}

		to, err := address.NewFromString(toAddr)
		if err != nil {
			return err
		}

		actor, err := api.StateGetActor(ctx, from, ts.Key())
		if err != nil {
			return err
		}

		nonce := actor.Nonce

		deadline := cctx.Uint64("deadline")
		partition := cctx.Uint64("partition")
		sectornos := cctx.Uint64Slice("sectors")
		newExpiration := cctx.Int64("new-expiration")
		new := cctx.Bool("new")

		sectors := bitfield.NewFromSet(sectornos)

		var method abi.MethodNum
		// todo: 智能选择方法
		var paramsByte []byte
		if !new {
			method = builtin.MethodsMiner.ExtendSectorExpiration

			var params *miner.ExtendSectorExpirationParams
			extensions := make([]miner.ExpirationExtension, 0)
			extensions = append(extensions, miner.ExpirationExtension{
				Deadline:      deadline,
				Partition:     partition,
				Sectors:       sectors,
				NewExpiration: abi.ChainEpoch(newExpiration),
			})

			params = &miner.ExtendSectorExpirationParams{
				Extensions: extensions,
			}

			paramsByte, err = SerializeParams(params)
			if err != nil {
				return err
			}
		} else {
			method = builtin.MethodsMiner.ExtendSectorExpiration2

			var params *miner.ExtendSectorExpiration2Params
			extensions := make([]miner.ExpirationExtension2, 0)
			extensions = append(extensions, miner.ExpirationExtension2{
				Deadline:          deadline,
				Partition:         partition,
				Sectors:           sectors,
				SectorsWithClaims: sectorClaims,
				NewExpiration:     abi.ChainEpoch(newExpiration),
			})

			params = &miner.ExtendSectorExpiration2Params{
				Extensions: extensions,
			}

			paramsByte, err = SerializeParams(params)
			if err != nil {
				return err
			}

		}

		msg := BuildMessage(from, to, types.NewInt(0), nonce, 1000000, abi.NewTokenAmount(1000000), abi.NewTokenAmount(5), method, paramsByte, false)

		mcid, err := PushMessage(ctx, api, &msg)
		if err != nil {
			return err
		}

		log.Infof("extend sector successfully, mcid: %v", mcid)

		return nil
	},
}

var extendClaimCmd = &cli.Command{
	Name: "extend-claim",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
		},
		&cli.StringFlag{
			Name: "token",
		},
		&cli.StringFlag{
			Name:  "tipset",
			Usage: "tipset key(pass comma separated array of cids), default chainhead",
		},
		&cli.StringFlag{
			Name:     "from",
			Required: true,
			Usage:    "message of from",
		},
		&cli.StringFlag{
			Name:     "provider",
			Required: true,
		},
		// todo: 多个deadline
		&cli.Int64Flag{
			Name:     "claimid",
			Required: true,
		},
		&cli.Int64Flag{
			Name:     "termmax",
			Required: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var requestHeader http.Header
		token := cctx.String("token")
		url := cctx.String("api-url")
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			return err
		}
		defer closer()

		fromAddr := cctx.String("from")
		providerAddr := cctx.String("provider")
		claimID := cctx.Int64("claimid")
		termMax := cctx.Int64("termmax")
		tipsetKey := cctx.String("tipset")

		ts, err := NewStringForTipSet(ctx, tipsetKey, api)
		if err != nil {
			return err
		}

		from, err := address.NewFromString(fromAddr)
		if err != nil {
			return err
		}

		provider, err := address.NewFromString(providerAddr)
		if err != nil {
			return err
		}

		actor, err := api.StateGetActor(ctx, from, ts.Key())
		if err != nil {
			return err
		}

		nonce := actor.Nonce

		providerID, err := address.IDFromAddress(provider)
		if err != nil {
			return err
		}

		method := builtin.MethodsVerifiedRegistry.ExtendClaimTerms

		var params *verifregtypes.ExtendClaimTermsParams
		terms := make([]verifregtypes.ClaimTerm, 0)
		terms = append(terms, verifregtypes.ClaimTerm{
			Provider: abi.ActorID(providerID),
			ClaimId:  verifregtypes.ClaimId(claimID),
			TermMax:  abi.ChainEpoch(termMax),
		})

		params = &verifregtypes.ExtendClaimTermsParams{
			Terms: terms,
		}

		paramsByte, err := SerializeParams(params)
		if err != nil {
			return err
		}

		msg := BuildMessage(from, provider, types.NewInt(0), nonce, 1000000, abi.NewTokenAmount(1000000), abi.NewTokenAmount(5), method, paramsByte, false)

		mcid, err := PushMessage(ctx, api, &msg)
		if err != nil {
			return err
		}

		log.Infof("extend sector successfully, mcid: %v", mcid)

		return nil
	},
}

const (
	BeginTime = "2020-08-25T06:00:00+08:00" // mainnet高度0时的时间
)

var BaseTime, _ = time.Parse(time.RFC3339, BeginTime)

func GetCurEpoch() abi.ChainEpoch {
	return abi.ChainEpoch((time.Now().Unix() - BaseTime.Unix()) / 30)
}

//// 续期程序
//// todo: log凑行数； 各种合理化的参数设置
//func ExtendExpiringSector(ctx context.Context, api v0api.FullNode, from, miner address.Address, feecap abi.TokenAmount, sectors []*lminer.SectorOnChainInfo, newExpiration abi.ChainEpoch) error {
//	actor, err := api.StateGetActor(ctx, from, types.EmptyTSK)
//	if err != nil {
//		return err
//	}
//
//	// method、params 分epoch讨论
//
//	var params *sminer.ExtendSectorExpirationParams
//	extensions := make([]sminer.ExpirationExtension, 0)
//	//store := adt.WrapStore(ctx, cbor.NewCborStore(blockstore.NewAPIBlockstore(api)))
//
//	locationMap := make(map[uint64]map[uint64]bitfield.BitField)
//	for _, sector := range sectors {
//		location, err := api.StateSectorPartition(ctx, miner, sector.SectorNumber, types.EmptyTSK)
//		if err != nil {
//			return err
//		}
//
//		if _, ok := locationMap[location.Deadline]; !ok {
//			locationMap[location.Deadline] = make(map[uint64]bitfield.BitField)
//			locationMap[location.Deadline][location.Partition] = bitfield.NewFromSet([]uint64{uint64(sector.SectorNumber)})
//		} else {
//			merge, err := bitfield.MergeBitFields(locationMap[location.Deadline][location.Partition], bitfield.NewFromSet([]uint64{uint64(sector.SectorNumber)}))
//			if err != nil {
//				return err
//			}
//
//			locationMap[location.Deadline][location.Partition] = merge
//		}
//	}
//
//	for deadline, partitionMap := range locationMap {
//		for partition, sectors := range partitionMap {
//			extensions = append(extensions, sminer.ExpirationExtension{
//				Deadline:      deadline,
//				Partition:     partition,
//				Sectors:       sectors,
//				NewExpiration: newExpiration,
//			})
//		}
//	}
//
//	params = &sminer.ExtendSectorExpirationParams{
//		Extensions: extensions,
//	}
//
//	paramsByte, err := SerializeParams(params)
//	if err != nil {
//		return err
//	}
//
//	msg := types.Message{
//		From:       from,
//		To:         miner,
//		Value:      types.NewInt(0),
//		Nonce:      actor.Nonce,
//		GasLimit:   1000000,
//		GasFeeCap:  feecap,
//		GasPremium: abi.NewTokenAmount(5),
//		Method:     builtin.MethodsMiner.ExtendSectorExpiration2,
//		Params:     paramsByte,
//	}
//
//	smsg, err := api.WalletSignMessage(ctx, msg.From, &msg)
//	if err != nil {
//		return err
//	}
//
//	cid, err := api.MpoolPush(ctx, smsg)
//	if err != nil {
//		return err
//	}
//
//	log.Infof("extend sector cid: %v", cid)
//	return nil
//}

func SerializeParams(i cbg.CBORMarshaler) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := i.MarshalCBOR(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func NewStringForTipSet(ctx context.Context, tipsetKeyStr string, api v0api.FullNode) (*types.TipSet, error) {
	if tipsetKeyStr == "" {
		return api.ChainHead(ctx)
	}

	tipsetKeyStrs := strings.Split(tipsetKeyStr, ",")

	var cids []cid.Cid
	for _, s := range tipsetKeyStrs {
		c, err := cid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	if len(cids) == 0 {
		return nil, nil
	}

	k := types.NewTipSetKey(cids...)
	ts, err := api.ChainGetTipSet(ctx, k)
	if err != nil {
		return nil, err
	}

	return ts, nil
}

func ParseSectorClaims(path string) ([]miner.SectorClaim, error) {
	file, err := os.Open(path)
	defer file.Close() //nolint:staticcheck
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// todo: 兼容其他版本
	var sectorClaims []miner.SectorClaim
	err = json.Unmarshal(bytes, &sectorClaims)
	if err != nil {
		return nil, err
	}

	return sectorClaims, nil
}

func PushMessage(ctx context.Context, api v0api.FullNode, msg *types.Message) (cid.Cid, error) {
	smsg, err := api.WalletSignMessage(ctx, msg.From, msg)
	if err != nil {
		return cid.Undef, err
	}

	mcid, err := api.MpoolPush(ctx, smsg)
	if err != nil {
		return cid.Undef, err
	}

	return mcid, nil
}

func BuildMessage(from, to address.Address, value abi.TokenAmount, nonce uint64, gasLimit int64, gasFeeCap, gasPremium abi.TokenAmount, method abi.MethodNum, params []byte, helper bool) types.Message {
	// todo: 专业性建议合理化参数
	if helper {
		// todo
		fmt.Println("todo")
	}

	msg := types.Message{
		From:       from,
		To:         to,
		Value:      types.NewInt(0),
		Nonce:      nonce,
		GasLimit:   gasLimit,
		GasFeeCap:  gasFeeCap,
		GasPremium: gasPremium,
		Method:     method,
		Params:     params,
	}

	return msg
}
