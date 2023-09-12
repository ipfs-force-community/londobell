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
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/go-state-types/builtin/v9/miner"
	verifregtypes "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	lminer "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/messagepool"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
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
			replaceCmd,
			mpoolFindCmd,
		},
		EnableBashCompletion: true,
	}

	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Errorf("cli error: %s", err)
		os.Exit(1)
	}
}

// alertCmd alert for expiring sectors and claims
var alertCmd = &cli.Command{
	Name: "alert",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
			Usage:    "address for api, e.g.: ws://127.0.0.1:1234/rpc/v0",
		},
		&cli.StringFlag{
			Name:  "token",
			Usage: "token for api",
		},
		&cli.StringFlag{
			Name:     "miner",
			Required: true,
			Usage:    "miner to monitor to alert",
		},
		&cli.Int64Flag{
			Name:  "dead-duration",
			Value: 12,
			Usage: "max interval between the expiration epoch and the current epoch",
		},
		&cli.DurationFlag{
			Name:  "tick",
			Usage: "time interval for alert, 24 * time.Hour default, e.ge:  --tick 1d",
		},
		&cli.StringFlag{
			Name:     "from-email",
			Required: true,
			Usage:    "email to send alert messqage",
		},
		&cli.StringFlag{
			Name:     "smtp-code",
			Required: true,
			Usage:    "authorization code of from email",
		},
		&cli.StringFlag{
			Name:     "to-email",
			Required: true,
			Usage:    "email to receive alert message",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		from := cctx.String("from-email")
		password := cctx.String("smtp-code")
		to := cctx.String("to")

		minerStr := cctx.String("miner")

		deadDuration := cctx.Int64("dead-duration")

		// email config
		smtpHost := "smtp.qq.com"
		smtpPort := 587
		auth := smtp.PlainAuth("", from, password, smtpHost)
		subject := "expiring sectors and claims alert"

		var requestHeader http.Header
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			log.Errorf("new fullnode %v failed: %v", url, err)
			return err
		}
		defer closer()

		miner, err := address.NewFromString(minerStr)
		if err != nil {
			log.Errorf("new miner %v failed: %v", minerStr, err)
			return err
		}

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
					log.Errorf("get sectors for miner %v failed: %v", miner, err)
					return err
				}

				expiringSectors := make([]*lminer.SectorOnChainInfo, 0)
				expiringClaims := make(map[verifregtypes.ClaimId]verifregtypes.Claim, 0)
				for _, sector := range sectors {
					if sector.Expiration > GetCurEpoch() && sector.Expiration-GetCurEpoch() < abi.ChainEpoch(deadDuration) {
						expiringSectors = append(expiringSectors, sector)
					}
				}

				outExpiringSectors, err := json.MarshalIndent(expiringSectors, "", "  ")
				if err != nil {
					log.Errorf("marshal expiringSectors failed: %v", err)
					return err
				}

				claimMap, err := api.StateGetClaims(ctx, miner, types.EmptyTSK)
				if err != nil {
					log.Errorf("get claims for miner %v failed: %v", miner, err)
					return err
				}

				for id, claim := range claimMap {
					// expired claims can also be renewed
					if claim.TermStart+claim.TermMax > GetCurEpoch() && claim.TermStart+claim.TermMax-GetCurEpoch() < abi.ChainEpoch(deadDuration) || claim.TermStart+claim.TermMax <= GetCurEpoch() {
						expiringClaims[id] = claim
					}
				}

				outExpiringClaims, err := json.MarshalIndent(expiringClaims, "", "  ")
				if err != nil {
					log.Errorf("marshal expiringClaims failed: %v", err)
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
					log.Errorf("send mail failed: %v, from: %v, to: %v, auth: %v", err, from, to, auth)
					continue
				}

				log.Info("Email sent successfully!")
			case <-ctx.Done():
				log.Infof("ctx done!!")
				return nil
			}
		}
	},
}

// queryCmd query sector or claim info specified
var queryCmd = &cli.Command{
	Name: "query",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
			Usage:    "address for api, e.g.: ws://127.0.0.1:1234/rpc/v0",
		},
		&cli.StringFlag{
			Name:  "token",
			Usage: "token for api",
		},
		&cli.StringFlag{
			Name:     "miner",
			Required: true,
			Usage:    "miner of sector or claim to query",
		},
		&cli.Int64Flag{
			Name:  "number",
			Usage: "number of sector or claim to query",
		},
		&cli.StringFlag{
			Name:  "tipset",
			Usage: "tipset key(pass comma separated array of cids), default chainhead",
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: "query type, sector or claim",
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		number := cctx.Int64("number")
		tipsetKey := cctx.String("tipset")
		minerStr := cctx.String("miner")

		qtype := cctx.String("type")

		var requestHeader http.Header
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			log.Errorf("new fullnode %v failed: %v", url, err)
			return err
		}
		defer closer()

		ts, err := NewStringForTipSet(ctx, tipsetKey, api)
		if err != nil {
			log.Errorf("new tipset failed: %v, tipsetKey: %v", err, tipsetKey)
			return err
		}

		miner, err := address.NewFromString(minerStr)
		if err != nil {
			log.Errorf("miner %v is invalid for query sector or claim: %v", minerStr, err)
			return err
		}

		switch qtype {
		case "sector":
			sectorInfo, err := api.StateSectorGetInfo(ctx, miner, abi.SectorNumber(number), ts.Key())
			if err != nil {
				log.Errorf("get sector info of sectornumber %v for miner %v failed: %v", abi.SectorNumber(number), miner, err)
				return err
			}

			formatSectorInfo, err := json.MarshalIndent(sectorInfo, "", "  ")
			if err != nil {
				log.Errorf("marshal sectorInfo failed: %v, err")
				return err
			}

			log.Infof("sectorInfo of %v for %v: %+v", miner, number, formatSectorInfo)
		case "claim":
			claim, err := api.StateGetClaim(ctx, miner, verifregtypes.ClaimId(number), ts.Key())
			if err != nil {
				log.Errorf("get claim of number %v for miner %v failed: %v", verifregtypes.ClaimId(number), miner, err)
				return err
			}

			formatClaim, err := json.MarshalIndent(claim, "", "  ")
			if err != nil {
				log.Errorf("marshal claim failed: %v", err)
				return err
			}

			log.Infof("claim of %v for %v: %+v", miner, number, formatClaim)
		default:
			return fmt.Errorf("invalid query type: %v", qtype)
		}

		return nil
	},
}

// extendSectorCmd extend sectors according to the params given or intelligent suggestion, need write permission for api
var extendSectorCmd = &cli.Command{
	Name: "extend-sector",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
			Usage:    "address for api, e.g.: ws://127.0.0.1:1234/rpc/v0",
		},
		&cli.StringFlag{
			Name:  "token",
			Usage: "token for api",
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
			Usage:    "miner for sectors extended",
		},
		&cli.StringFlag{
			Name:  "extensions",
			Usage: "json to store params of extend sectors, e.g.: ",
			// new expirations of alerting sectors are decided by user
			// 建议
		},
		&cli.Uint64Flag{
			Name:  "value",
			Usage: "value for new message (attoFIL/GasUnit)",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "gas limit for new message (GasUnit)",
		},
		&cli.Int64Flag{
			Name:  "gas-feecap",
			Usage: "gas feecap for new message (burn and pay to miner, attoFIL/GasUnit)",
		},
		&cli.Int64Flag{
			Name:  "gas-premium",
			Usage: "gas price for new message (pay to miner, attoFIL/GasUnit)",
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		fromAddr := cctx.String("from")
		toAddr := cctx.String("to")
		tipsetKey := cctx.String("tipset")

		extensions := cctx.String("extensions")

		value := cctx.Uint64("value")
		gasLimit := cctx.Int64("gas-limit")
		gasFeeCap := cctx.Int64("gas-feecap")
		gasPremium := cctx.Int64("gas-premium")

		var requestHeader http.Header
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			log.Errorf("new fullnode %v failed: %v", url, err)
			return err
		}
		defer closer()

		ts, err := NewStringForTipSet(ctx, tipsetKey, api)
		if err != nil {
			log.Errorf("new tipset failed: %v, tipsetKey: %v", err, tipsetKey)
			return err
		}

		from, err := address.NewFromString(fromAddr)
		if err != nil {
			log.Errorf("new from address %v failed: %v", fromAddr, err)
			return err
		}

		to, err := address.NewFromString(toAddr)
		if err != nil {
			log.Errorf("new to address %v failed: %v", toAddr, err)
			return err
		}

		actor, err := api.StateGetActor(ctx, from, ts.Key())
		if err != nil {
			log.Errorf("get actor %v at ts %v failed: %v", from, ts.Key(), err)
			return err
		}

		nonce := actor.Nonce

		extendSectorExpiration2Params, err := ParseSectorExtensions(extensions)
		if err != nil {
			log.Errorf("parse sector extensions %v failed: %v", extensions, err)
			return err
		}

		// choose extend sector method intelligently
		commitLegacy, err := IsExtendCommitLegacy(ctx, extendSectorExpiration2Params, to, api, ts)
		if err != nil {
			log.Errorf("adjust extend method failed: %v", err)
			return err
		}

		var (
			method     abi.MethodNum
			paramsByte []byte
		)
		if commitLegacy {
			method = builtin.MethodsMiner.ExtendSectorExpiration

			extensions := make([]miner.ExpirationExtension, 0)
			for _, ex2 := range extendSectorExpiration2Params.Extensions {
				extensions = append(extensions, miner.ExpirationExtension{
					Deadline:      ex2.Deadline,
					Partition:     ex2.Partition,
					Sectors:       ex2.Sectors,
					NewExpiration: ex2.NewExpiration,
				})
			}

			params := &miner.ExtendSectorExpirationParams{
				Extensions: extensions,
			}

			paramsByte, err = SerializeParams(params)
			if err != nil {
				log.Errorf("serialize params %v failed: %v", params, err)
				return err
			}
		} else {
			method = builtin.MethodsMiner.ExtendSectorExpiration2

			// add sectorclaims intelligently
			err = FillSectorsWithClaims(ctx, &extendSectorExpiration2Params, to, api, ts)
			if err != nil {
				log.Errorf("fill sectorclaims for sectors failed: %v", err)
				return err
			}

			paramsByte, err = SerializeParams(&extendSectorExpiration2Params)
			if err != nil {
				log.Errorf("serialize params %v failed: %v", extendSectorExpiration2Params, err)
				return err
			}
		}

		helper := false
		if gasLimit == 0 || gasFeeCap == 0 || gasPremium == 0 {
			helper = true
		}

		// support custom message & intelligently help evaluate message gas parameters
		msg, err := BuildMessage(ctx, api, ts, from, to, types.NewInt(value), nonce, gasLimit, abi.NewTokenAmount(gasFeeCap), abi.NewTokenAmount(gasPremium), method, paramsByte, helper)
		if err != nil {
			log.Errorf("build message failed: %v", err)
			return err
		}

		mcid, err := PushMessage(ctx, api, msg)
		if err != nil {
			log.Errorf("put message %+v failed: %v", *msg, err)
			return err
		}

		log.Infof("extend sector successfully, mcid: %v", mcid)

		return nil
	},
}

// extendClaimCmd extend claims according to the params given or intelligent suggestion, need write permission for api
var extendClaimCmd = &cli.Command{
	Name: "extend-claim",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
			Usage:    "address for api, e.g.: ws://127.0.0.1:1234/rpc/v0",
		},
		&cli.StringFlag{
			Name:  "token",
			Usage: "token for api",
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
			Usage:    "provider for claims extended",
		},
		&cli.StringFlag{
			Name:     "claimTerm",
			Required: true,
			Usage:    "json of ClaimTerm, Set term_max as large as possible",
		},
		&cli.Uint64Flag{
			Name:  "value",
			Usage: "value for new message (attoFIL/GasUnit)",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "gas limit for new message (GasUnit)",
		},
		&cli.Int64Flag{
			Name:  "gas-feecap",
			Usage: "gas feecap for new message (burn and pay to miner, attoFIL/GasUnit)",
		},
		&cli.Int64Flag{
			Name:  "gas-premium",
			Usage: "gas price for new message (pay to miner, attoFIL/GasUnit)",
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		fromAddr := cctx.String("from")
		providerAddr := cctx.String("provider")
		tipsetKey := cctx.String("tipset")

		claimTermJSON := cctx.String("claimTerm")

		value := cctx.Uint64("value")
		gasLimit := cctx.Int64("gas-limit")
		gasFeeCap := cctx.Int64("gas-feecap")
		gasPremium := cctx.Int64("gas-premium")

		var requestHeader http.Header
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			log.Errorf("new fullnode %v failed: %v", url, err)
			return err
		}
		defer closer()

		ts, err := NewStringForTipSet(ctx, tipsetKey, api)
		if err != nil {
			log.Errorf("new tipset failed: %v, tipsetKey: %v", err, tipsetKey)
			return err
		}

		from, err := address.NewFromString(fromAddr)
		if err != nil {
			log.Errorf("new from address %v failed: %v", fromAddr, err)
			return err
		}

		provider, err := address.NewFromString(providerAddr)
		if err != nil {
			log.Errorf("new provider address %v failed: %v", providerAddr, err)
			return err
		}

		actor, err := api.StateGetActor(ctx, from, ts.Key())
		if err != nil {
			log.Errorf("get actor %v at ts %v failed: %v", from, ts.Key(), err)
			return err
		}

		nonce := actor.Nonce

		claimTerms, err := ParseClaimTerm(claimTermJSON)
		if err != nil {
			log.Errorf("parse claim term %v failed: %v", claimTermJSON, err)
			return err
		}

		method := builtin.MethodsVerifiedRegistry.ExtendClaimTerms

		var params *verifregtypes.ExtendClaimTermsParams
		params = &verifregtypes.ExtendClaimTermsParams{
			Terms: claimTerms,
		}

		paramsByte, err := SerializeParams(params)
		if err != nil {
			log.Errorf("serialize params %v failed: %v", params, err)
			return err
		}

		helper := false
		if gasLimit == 0 || gasFeeCap == 0 || gasPremium == 0 {
			helper = true
		}

		// support custom message & intelligently help evaluate message gas parameters
		msg, err := BuildMessage(ctx, api, ts, from, provider, types.NewInt(value), nonce, gasLimit, abi.NewTokenAmount(gasFeeCap), abi.NewTokenAmount(gasPremium), method, paramsByte, helper)
		if err != nil {
			log.Errorf("build message failed: %v", err)
			return err
		}

		mcid, err := PushMessage(ctx, api, msg)
		if err != nil {
			log.Errorf("put message %+v failed: %v", *msg, err)
			return err
		}

		log.Infof("extend sector successfully, mcid: %v", mcid)

		return nil
	},
}

// replaceCmd replace a message in the mempool
var replaceCmd = &cli.Command{
	Name:  "replace",
	Usage: "replace extend message in the mpool to be packed early",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
		},
		&cli.StringFlag{
			Name: "token",
		},
		&cli.StringFlag{
			Name:  "gas-feecap",
			Usage: "gas feecap for new message (burn and pay to miner, attoFIL/GasUnit)",
		},
		&cli.StringFlag{
			Name:  "gas-premium",
			Usage: "gas price for new message (pay to miner, attoFIL/GasUnit)",
		},
		&cli.Int64Flag{
			Name:  "gas-limit",
			Usage: "gas limit for new message (GasUnit)",
		},
		&cli.BoolFlag{
			Name:  "auto",
			Usage: "automatically reprice the specified message",
		},
		&cli.StringFlag{
			Name:  "fee-limit",
			Usage: "Spend up to X FIL for this message in units of FIL. Previously when flag was `max-fee` units were in attoFIL. Applicable for auto mode",
		},
	},
	ArgsUsage: "<from> <nonce> | <message-cid>",
	Action: func(cctx *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		var requestHeader http.Header
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			return err
		}
		defer closer()

		var from address.Address
		var nonce uint64
		switch cctx.NArg() {
		case 1:
			mcid, err := cid.Decode(cctx.Args().First())
			if err != nil {
				return err
			}

			msg, err := api.ChainGetMessage(ctx, mcid)
			if err != nil {
				return xerrors.Errorf("could not find referenced message: %w", err)
			}

			from = msg.From
			nonce = msg.Nonce
		case 2:
			arg0 := cctx.Args().Get(0)
			f, err := address.NewFromString(arg0)
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(cctx.Args().Get(1), 10, 64)
			if err != nil {
				return err
			}

			from = f
			nonce = n
		default:
			return cli.ShowCommandHelp(cctx, cctx.Command.Name)
		}

		ts, err := api.ChainHead(ctx)
		if err != nil {
			return xerrors.Errorf("getting chain head: %w", err)
		}

		pending, err := api.MpoolPending(ctx, ts.Key())
		if err != nil {
			return err
		}

		var found *types.SignedMessage
		for _, p := range pending {
			if p.Message.From == from && p.Message.Nonce == nonce {
				found = p
				break
			}
		}

		if found == nil {
			return xerrors.Errorf("no pending message found from %s with nonce %d", from, nonce)
		}

		msg := found.Message

		if cctx.Bool("auto") {
			cfg, err := api.MpoolGetConfig(ctx)
			if err != nil {
				return xerrors.Errorf("failed to lookup the message pool config: %w", err)
			}

			defaultRBF := messagepool.ComputeRBF(msg.GasPremium, cfg.ReplaceByFeeRatio)

			var mss *lapi.MessageSendSpec
			if cctx.IsSet("fee-limit") {
				maxFee, err := types.ParseFIL(cctx.String("fee-limit"))
				if err != nil {
					return xerrors.Errorf("parsing max-spend: %w", err)
				}
				mss = &lapi.MessageSendSpec{
					MaxFee: abi.TokenAmount(maxFee),
				}
			}

			// msg.GasLimit = 0 // TODO: need to fix the way we estimate gas limits to account for the messages already being in the mempool
			msg.GasFeeCap = abi.NewTokenAmount(0)
			msg.GasPremium = abi.NewTokenAmount(0)
			retm, err := api.GasEstimateMessageGas(ctx, &msg, mss, types.EmptyTSK)
			if err != nil {
				return xerrors.Errorf("failed to estimate gas values: %w", err)
			}

			msg.GasPremium = big.Max(retm.GasPremium, defaultRBF)
			msg.GasFeeCap = big.Max(retm.GasFeeCap, msg.GasPremium)

			mff := func() (abi.TokenAmount, error) {
				return abi.TokenAmount(config.DefaultDefaultMaxFee), nil
			}

			messagepool.CapGasFee(mff, &msg, mss)
		} else {
			if cctx.IsSet("gas-limit") {
				msg.GasLimit = cctx.Int64("gas-limit")
			}
			msg.GasPremium, err = types.BigFromString(cctx.String("gas-premium"))
			if err != nil {
				return xerrors.Errorf("parsing gas-premium: %w", err)
			}
			// TODO: estimate fee cap here
			msg.GasFeeCap, err = types.BigFromString(cctx.String("gas-feecap"))
			if err != nil {
				return xerrors.Errorf("parsing gas-feecap: %w", err)
			}
		}

		smsg, err := api.WalletSignMessage(ctx, msg.From, &msg)
		if err != nil {
			return xerrors.Errorf("failed to sign message: %w", err)
		}

		cid, err := api.MpoolPush(ctx, smsg)
		if err != nil {
			return xerrors.Errorf("failed to push new message to mempool: %w", err)
		}

		log.Infof("new message cid: %v", cid)
		return nil
	},
}

// mpoolFindCmd find a message in the mempool
var mpoolFindCmd = &cli.Command{
	Name:  "find",
	Usage: "find a message in the mempool",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-url",
			Required: true,
		},
		&cli.StringFlag{
			Name: "token",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "search for messages with given 'from' address",
		},
		&cli.StringFlag{
			Name:  "to",
			Usage: "search for messages with given 'to' address",
		},
		&cli.Int64Flag{
			Name:  "method",
			Usage: "search for messages with given method",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		var requestHeader http.Header
		if token != "" {
			requestHeader = http.Header{"Authorization": []string{"Bearer " + token}}
		}

		api, closer, err := client.NewFullNodeRPCV0(ctx, url, requestHeader)
		if err != nil {
			return err
		}
		defer closer()

		pending, err := api.MpoolPending(ctx, types.EmptyTSK)
		if err != nil {
			return err
		}

		var toFilter, fromFilter address.Address
		if cctx.IsSet("to") {
			a, err := address.NewFromString(cctx.String("to"))
			if err != nil {
				return xerrors.Errorf("'to' address was invalid: %w", err)
			}

			toFilter = a
		}

		if cctx.IsSet("from") {
			a, err := address.NewFromString(cctx.String("from"))
			if err != nil {
				return xerrors.Errorf("'from' address was invalid: %w", err)
			}

			fromFilter = a
		}

		var methodFilter *abi.MethodNum
		if cctx.IsSet("method") {
			m := abi.MethodNum(cctx.Int64("method"))
			methodFilter = &m
		}

		var out []*types.SignedMessage
		for _, m := range pending {
			if toFilter != address.Undef && m.Message.To != toFilter {
				continue
			}

			if fromFilter != address.Undef && m.Message.From != fromFilter {
				continue
			}

			if methodFilter != nil && *methodFilter != m.Message.Method {
				continue
			}

			out = append(out, m)
		}

		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}

		log.Info(string(b))
		return nil
	},
}

const (
	// mainnet高度0时的时间
	BeginTime = "2020-08-25T06:00:00+08:00"
)

var BaseTime, _ = time.Parse(time.RFC3339, BeginTime)

func GetCurEpoch() abi.ChainEpoch {
	return abi.ChainEpoch((time.Now().Unix() - BaseTime.Unix()) / 30)
}

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

// todo: bitfiled json怎么传入
func ParseSectorExtensions(path string) (miner.ExtendSectorExpiration2Params, error) {
	file, err := os.Open(path)
	defer file.Close() //nolint:staticcheck
	if err != nil {
		return miner.ExtendSectorExpiration2Params{}, err
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return miner.ExtendSectorExpiration2Params{}, err
	}

	// todo: 兼容其他版本
	var extendSectorExpiration2Params miner.ExtendSectorExpiration2Params
	err = json.Unmarshal(bytes, &extendSectorExpiration2Params)
	if err != nil {
		return miner.ExtendSectorExpiration2Params{}, err
	}

	return extendSectorExpiration2Params, nil
}

func ParseClaimTerm(path string) ([]verifregtypes.ClaimTerm, error) {
	file, err := os.Open(path)
	defer file.Close() //nolint:staticcheck
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	claimTerms := make([]verifregtypes.ClaimTerm, 0)
	err = json.Unmarshal(bytes, &claimTerms)
	if err != nil {
		return nil, err
	}

	return claimTerms, nil
}

// IsExtendCommitLegacy return whether to use legacy extend method
func IsExtendCommitLegacy(ctx context.Context, extendSectorExpiration2Params miner.ExtendSectorExpiration2Params, miner address.Address, api v0api.FullNode, tipset *types.TipSet) (bool, error) {
	var commitLegacy = true
	for _, extension := range extendSectorExpiration2Params.Extensions {
		err := extension.Sectors.ForEach(func(u uint64) error {
			if commitLegacy == false {
				return nil
			}

			sectorInfo, err := api.StateSectorGetInfo(ctx, miner, abi.SectorNumber(u), tipset.Key())
			if err != nil {
				return err
			}

			if sectorInfo.SimpleQAPower && (sectorInfo.VerifiedDealWeight.GreaterThan(big.NewInt(0)) || sectorInfo.DealWeight.GreaterThanEqual(big.NewInt(0))) {
				commitLegacy = false
				return nil
			}

			return nil
		})

		if err != nil {
			return commitLegacy, err
		}
	}

	return commitLegacy, nil
}

var errFailedFillSectorsWithClaims = fmt.Errorf("failed fill sectors with claims")

// todo: 或者可以帮用户选择合适的new_expiration?
// FillSectorsWithClaims fill sectorclaim for sectors extended, include claimids maintained or dropped. Failed if there are claims needed to be dropped but cannot be dropped
func FillSectorsWithClaims(ctx context.Context, extendSectorExpiration2Params *miner.ExtendSectorExpiration2Params, provider address.Address, api v0api.FullNode, tipset *types.TipSet) error {
	failed := false

	claimsForSectorsMap, err := ClaimsForSectors(ctx, provider, api, tipset)
	if err != nil {
		return err
	}

	extensions := extendSectorExpiration2Params.Extensions
	for i, ex2 := range extensions {
		if failed {
			return errFailedFillSectorsWithClaims
		}

		newExpiration := ex2.NewExpiration
		sectorsWithClaims := make([]miner.SectorClaim, 0)
		err := ex2.Sectors.ForEach(func(u uint64) error {
			if failed {
				return errFailedFillSectorsWithClaims
			}

			// bind claim and sector
			claims := claimsForSectorsMap[abi.SectorNumber(u)]

			maintainClaims, dropClaims := MaintainAndDropClaims(newExpiration, claims)
			if len(dropClaims) > 0 {
				sectorInfo, err := api.StateSectorGetInfo(ctx, provider, abi.SectorNumber(u), tipset.Key())
				if err != nil {
					return err
				}

				canDropClaims := CanDropClaims(sectorInfo.Expiration, tipset.Height())
				if !canDropClaims {
					// new_expiration is too large or extend claims first
					log.Warnf("claims is not allowed to drop for expiration(%v)-curEpoch(%v) <= verifregtypes.EndOfLifeClaimDropPeriod(30d), new expiration %v is too high than claim.term_start + claim.term_max, dropClaims: %v", sectorInfo.Expiration, tipset.Key(), newExpiration, dropClaims)
					failed = true
				}
			}

			if failed {
				return errFailedFillSectorsWithClaims
			}

			sectorsWithClaims = append(sectorsWithClaims, miner.SectorClaim{
				SectorNumber:   abi.SectorNumber(u),
				MaintainClaims: maintainClaims,
				DropClaims:     dropClaims,
			})

			return nil
		})

		if err != nil {
			return err
		}

		extendSectorExpiration2Params.Extensions[i].SectorsWithClaims = sectorsWithClaims
	}

	return nil
}

// ClaimsForSectors bind sector and corresponding claims
func ClaimsForSectors(ctx context.Context, provider address.Address, api v0api.FullNode, tipset *types.TipSet) (map[abi.SectorNumber]map[verifregtypes.ClaimId]verifregtypes.Claim, error) {
	claimsForSectorsMap := make(map[abi.SectorNumber]map[verifregtypes.ClaimId]verifregtypes.Claim)
	claims, err := api.StateGetClaims(ctx, provider, tipset.Key())
	if err != nil {
		return nil, err
	}

	for id, claim := range claims {
		if _, ok := claimsForSectorsMap[claim.Sector]; !ok {
			claimMap := make(map[verifregtypes.ClaimId]verifregtypes.Claim)
			claimMap[id] = claim
			claimsForSectorsMap[claim.Sector] = claimMap
		} else {
			claimsForSectorsMap[claim.Sector][id] = claim
		}
	}

	return claimsForSectorsMap, nil
}

// maintain: decl.new_expiration <= claim.term_start + claim.term_max
// drop: sector.expiration - curr_epoch <= policy.end_of_life_claim_drop_period
func MaintainAndDropClaims(newExpiration abi.ChainEpoch, claims map[verifregtypes.ClaimId]verifregtypes.Claim) ([]verifregtypes.ClaimId, []verifregtypes.ClaimId) {
	maintainClaims, dropClaims := make([]verifregtypes.ClaimId, 0), make([]verifregtypes.ClaimId, 0)
	for id, claim := range claims {
		if newExpiration > claim.TermStart+claim.TermMax {
			dropClaims = append(dropClaims, id)
		} else {
			maintainClaims = append(maintainClaims, id)
		}
	}

	return maintainClaims, dropClaims
}

// CanDropClaims return whether claims of sector can be dropped
func CanDropClaims(expiration, curEpoch abi.ChainEpoch) bool {
	return expiration-curEpoch <= verifregtypes.EndOfLifeClaimDropPeriod
}

// todo: 专业性建议合理化参数
// gaslimit, gasfeecap, gaspremium,
// gaslimit和gasused相关
// gasfeecap和basefee相关，大点就行，差距比gasPremium大
// gaspremium和其他用户给的gaspremium相关，允许replace
func BuildMessage(ctx context.Context, api v0api.FullNode, ts *types.TipSet, from, to address.Address, value abi.TokenAmount, nonce uint64, gasLimit int64, gasFeeCap, gasPremium abi.TokenAmount, method abi.MethodNum, params []byte, helper bool) (*types.Message, error) {
	if helper {
		// estimate gasLimit、gasFeeCap、gasPremium
		msg := &types.Message{
			Version:    0,
			To:         to,
			From:       from,
			Nonce:      nonce,
			Value:      value,
			GasLimit:   0,
			GasFeeCap:  abi.NewTokenAmount(0),
			GasPremium: abi.NewTokenAmount(0),
			Method:     method,
			Params:     params,
		}

		retm, err := api.GasEstimateMessageGas(ctx, msg, nil, ts.Key())
		if err != nil {
			return nil, err
		}

		return retm, nil
	}

	msg := &types.Message{
		From:       from,
		To:         to,
		Value:      value,
		Nonce:      nonce,
		GasLimit:   gasLimit,
		GasFeeCap:  gasFeeCap,
		GasPremium: gasPremium,
		Method:     method,
		Params:     params,
	}

	return msg, nil
}

// PushMessage send message to chain
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
