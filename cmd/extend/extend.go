package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/smtp"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/lotus/lib/lotuslog"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	sbuiltin "github.com/filecoin-project/go-state-types/builtin"
	smath "github.com/filecoin-project/go-state-types/builtin/v11/util/math"
	"github.com/filecoin-project/go-state-types/builtin/v9/miner"
	verifregtypes "github.com/filecoin-project/go-state-types/builtin/v9/verifreg"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/build"
	lminer "github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/messagepool"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/node/config"
	"github.com/filecoin-project/specs-actors/v8/actors/util/smoothing"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"
)

var log = logging.Logger("extend")

// todo: params参数推荐 unproven alert
func main() {
	lotuslog.SetupLogLevels()
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
			CheckUnprovenCmd,
			DrawRewardData,
			ComputeRewardData,
			Cal,
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
		to := cctx.String("to-email")

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
					continue
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
					continue
				}

				claimMap, err := api.StateGetClaims(ctx, miner, types.EmptyTSK)
				if err != nil {
					log.Errorf("get claims for miner %v failed: %v", miner, err)
					continue
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
					continue
				}

				// send email
				body := ""
				if len(expiringSectors) > 0 {
					body += fmt.Sprintf("expiring sectors: %+v", string(outExpiringSectors))
				}
				if len(expiringClaims) > 0 {
					body += fmt.Sprintf("expiring claims: %+v", string(outExpiringClaims))
					log.Infof("expiring claims: %+v", string(outExpiringClaims))
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

				log.Infof("Email sent successfully!")
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

			log.Infof("sectorInfo of %v for %v:\n %+v", miner, number, string(formatSectorInfo))
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

			log.Infof("claim of %v for %v:\n %+v", number, miner, string(formatClaim))
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
			Name:  "path",
			Usage: "json to store params of extend sectors, e.g.: ",
			// new expirations of alerting sectors are decided by user
			// 建议
		},
		&cli.BoolFlag{
			Name:  "help_expiration",
			Usage: "allow empty new expiration, system will fill max new expiration for you if true",
			Value: true,
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
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "send extend message if true, false default",
			Value: false,
		},
		&cli.Uint64Flag{
			Name:  "confidence",
			Usage: "number of block confirmations to wait for",
			Value: build.MessageConfidence,
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		fromAddr := cctx.String("from")
		toAddr := cctx.String("miner")
		tipsetKey := cctx.String("tipset")

		path := cctx.String("path")
		helpExpiration := cctx.Bool("help_expiration")

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

		// deadline, partition自动填
		extendSectorExpiration2Params, err := ParseSectorExtensions(ctx, path, to, api, ts, helpExpiration)
		if err != nil {
			log.Errorf("parse sector path %v failed: %v", path, err)
			return err
		}

		//// choose extend sector method intelligently
		//commitLegacy, err := IsExtendCommitLegacy(ctx, *extendSectorExpiration2Params, to, api, ts)
		//if err != nil {
		//	log.Errorf("adjust extend method failed: %v", err)
		//	return err
		//}

		var (
			method     abi.MethodNum
			paramsByte []byte
		)

		method = builtin.MethodsMiner.ExtendSectorExpiration2

		// add sectorclaims intelligently
		err = FillSectorsWithClaims(ctx, extendSectorExpiration2Params, to, api, ts)
		if err != nil {
			log.Errorf("fill sectorclaims for sectors failed: %v", err)
			return err
		}

		paramsByte, err = SerializeParams(extendSectorExpiration2Params)
		if err != nil {
			log.Errorf("serialize params %v failed: %v", extendSectorExpiration2Params, err)
			return err
		}

		helpeMessage := false
		if gasLimit == 0 || gasFeeCap == 0 || gasPremium == 0 {
			helpeMessage = true
		}

		// support custom message & intelligently help evaluate message gas parameters
		msg, err := BuildMessage(ctx, api, ts, from, to, types.NewInt(value), nonce, gasLimit, abi.NewTokenAmount(gasFeeCap), abi.NewTokenAmount(gasPremium), method, paramsByte, helpeMessage)
		if err != nil {
			log.Errorf("build message failed: %v", err)
			return err
		}

		log.Infof("extend sector message will be sent by %s for %s", msg.From.String(), msg.To.String())
		log.Infof("extend gas params, value: %v, gaslimit: %v, gasfeecap: %v, gaspremium: %v", msg.Value, msg.GasLimit, msg.GasFeeCap, msg.GasPremium)
		log.Infof("there are %v groups of sectors will be extended: ", len(extendSectorExpiration2Params.Extensions))
		log.Infof("deadline\tpartition\tsectors\tnew_expiration\t")
		for _, extension := range extendSectorExpiration2Params.Extensions {
			sectors, err := extension.Sectors.All(miner.AddressedSectorsMax)
			if err != nil {
				log.Errorf("get extension sectors failed: %v", err)
				return err
			}

			log.Infof("%v\t%v\t%v\t%v\t", extension.Deadline, extension.Partition, sectors, extension.NewExpiration)
		}

		if !cctx.Bool("really-do-it") {
			log.Warnf("will not send extend message when really-do-it false")
			return nil
		}

		mcid, err := PushMessage(ctx, api, msg, cctx.Uint64("confidence"))
		if err != nil {
			log.Errorf("push message %+v failed: %v", *msg, err)
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
			Name:     "path",
			Required: true,
			Usage:    "json of ClaimTerm, Set term_max as large as possible",
		},
		&cli.BoolFlag{
			Name:  "help_expiration",
			Usage: "allow empty new expiration, system will fill max new expiration for you if true",
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
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "send extend message if true, false default",
			Value: false,
		},
		&cli.Uint64Flag{
			Name:  "confidence",
			Usage: "number of block confirmations to wait for",
			Value: build.MessageConfidence,
		},
	},
	Action: func(cctx *cli.Context) error {
		// query sector or claim for miner
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		token := cctx.String("token")
		url := cctx.String("api-url")

		fromAddr := cctx.String("from")
		tipsetKey := cctx.String("tipset")

		path := cctx.String("path")
		helpExpiration := cctx.Bool("help_expiration")
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

		actor, err := api.StateGetActor(ctx, from, ts.Key())
		if err != nil {
			log.Errorf("get actor %v at ts %v failed: %v", from, ts.Key(), err)
			return err
		}

		nonce := actor.Nonce

		claimTerms, err := ParseClaimTerm(ctx, from, path, api, helpExpiration)
		if err != nil {
			log.Errorf("parse claim term %v failed: %v", path, err)
			return err
		}

		method := builtin.MethodsVerifiedRegistry.ExtendClaimTerms
		to := sbuiltin.VerifiedRegistryActorAddr

		var params *verifregtypes.ExtendClaimTermsParams
		params = &verifregtypes.ExtendClaimTermsParams{
			Terms: claimTerms,
		}

		paramsByte, err := SerializeParams(params)
		if err != nil {
			log.Errorf("serialize params %v failed: %v", params, err)
			return err
		}

		helpMessage := false
		if gasLimit == 0 || gasFeeCap == 0 || gasPremium == 0 {
			helpMessage = true
		}

		// support custom message & intelligently help evaluate message gas parameters
		msg, err := BuildMessage(ctx, api, ts, from, to, types.NewInt(value), nonce, gasLimit, abi.NewTokenAmount(gasFeeCap), abi.NewTokenAmount(gasPremium), method, paramsByte, helpMessage)
		if err != nil {
			log.Errorf("build message failed: %v", err)
			return err
		}

		log.Infof("extend claim message will be sent by %s to %s", msg.From.String(), msg.To.String())
		log.Infof("extend gas params, value: %v, gaslimit: %v, gasfeecap: %v, gaspremium: %v", msg.Value, msg.GasLimit, msg.GasFeeCap, msg.GasPremium)
		log.Infof("there are %v groups of claims will be extended: ", len(params.Terms))
		log.Infof("provider\tclaimId\tterm_max\t")
		for _, term := range params.Terms {
			log.Infof("%v\t%v\t%v\t", term.Provider, term.ClaimId, term.TermMax)
		}

		if !cctx.Bool("really-do-it") {
			log.Warnf("will not send extend message when really-do-it false")
			return nil
		}

		mcid, err := PushMessage(ctx, api, msg, cctx.Uint64("confidence"))
		if err != nil {
			log.Errorf("push message %+v failed: %v", *msg, err)
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

var CheckUnprovenCmd = &cli.Command{
	Name:    "check-unproven",
	Aliases: []string{"c"},
	Usage:   "check precommit info of miner",
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
			Name: "miner",
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
			log.Errorf("new fullnode %v failed: %v", url, err)
			return err
		}
		defer closer()

		addr, err := address.NewFromString(cctx.String("miner"))
		if err != nil {
			return err
		}

		ts, err := api.ChainHead(ctx)
		if err != nil {
			return err
		}

		act, err := api.StateGetActor(ctx, addr, ts.Key())
		if err != nil {
			return fmt.Errorf("loading state error: %+v", err)
		}

		stor := store.ActorStore(ctx, blockstore.NewAPIBlockstore(api))

		state, err := lminer.Load(stor, act)
		if err != nil {
			return fmt.Errorf("loading miner state: %+v", err)
		}

		type sector struct {
			sectorID abi.SectorNumber
			active   abi.ChainEpoch
			expire   abi.ChainEpoch
		}
		ss := []sector{}
		_ = state.ForEachPrecommittedSector(func(info miner.SectorPreCommitOnChainInfo) error {
			ss = append(ss, sector{
				sectorID: info.Info.SectorNumber,
				active:   info.PreCommitEpoch,
				expire:   info.PreCommitEpoch + 150 + 2880*30,
			})
			return nil
		})

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].expire < ss[j].expire
		})

		for i := range ss {
			fmt.Printf("%d %v %v\n", ss[i].sectorID, EpochToTime(ss[i].active), EpochToTime(ss[i].expire))
		}

		return nil
	},
}

var DrawRewardData = &cli.Command{
	Name:    "draw-reward-data",
	Aliases: []string{"c"},
	Usage:   "draw reward data and base line data",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "day",
			Value: 365,
		},
	},
	Action: func(context *cli.Context) error {
		api, closer, err := cliutil.GetFullNodeAPI(context)
		if err != nil {
			return err
		}
		defer closer()

		ts, err := api.ChainHead(context.Context)
		if err != nil {
			return err
		}

		ms, err := api.StateListMiners(context.Context, ts.Key())
		if err != nil {
			return err
		}

		day := context.Int64("day")

		cnt := make([]int64, day, day)

		for i := range ms {
			fmt.Println("start deal miner", ms[i])

			s, err := api.StateMinerActiveSectors(context.Context, ms[i], ts.Key())
			if err != nil {
				return err
			}

			for j := range s {
				if s[j] == nil {
					fmt.Println("meet nil pointer ", j)
					continue
				}
				if s[j].Expiration < abi.ChainEpoch(day*2880)+ts.Height() && s[j].VerifiedDealWeight.Equals(big.Zero()) {
					if s[j].SealProof == 8 {
						cnt[int(s[j].Expiration-ts.Height())/2880]++
					} else {
						cnt[int(s[j].Expiration-ts.Height())/2880] += 2
					}
				}
			}
			fmt.Println("end deal miner", ms[i])
		}

		for i := range cnt {
			fmt.Println(cnt[i])
		}

		return nil
	},
}

var ComputeRewardData = &cli.Command{
	Name:    "compute-reward-data",
	Aliases: []string{"c"},
	Usage:   "draw reward data and base line data",
	Flags:   []cli.Flag{},
	Action: func(context *cli.Context) error {
		s := strings.Split(bad, "\n")
		fmt.Println(len(s))
		ctx := context.Context
		api, closer, err := cliutil.GetFullNodeAPI(context)
		if err != nil {
			return err
		}
		defer closer()
		addr, err := address.NewFromString("f02")
		if err != nil {
			return err
		}

		actorstate, err := api.StateReadState(ctx, addr, types.EmptyTSK)
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(actorstate.State, "", "  ")
		if err != nil {
			return err
		}

		st := &State{}
		err = json.Unmarshal(data, st)
		if err != nil {
			return err
		}

		addrPower, err := address.NewFromString("f04")
		if err != nil {
			return err
		}

		powerActorstate, err := api.StateReadState(ctx, addrPower, types.EmptyTSK)
		if err != nil {
			return err
		}
		data, err = json.MarshalIndent(powerActorstate.State, "", "  ")
		if err != nil {
			return err
		}

		circulatingSupply := big.MustFromString("409044739220948065033492596")
		//fmt.Println(string(data))
		st1 := &PowerState{}
		err = json.Unmarshal(data, st1)
		if err != nil {
			return err
		}

		fmt.Println("init", InitBaselinePower())
		fmt.Println(st)
		all := st1.ThisEpochRawBytePower

		cnt1, cnt2, cnt3, cnt4, cnt5 := []big.Int{}, []big.Int{}, []big.Int{}, []big.Int{}, []big.Int{}
		for i := range s {
			sint, err := strconv.ParseInt(s[i], 10, 64)
			if err != nil {
				return err
			}
			//_ = i
			sszie := big.NewInt((sint - 163840) * 32 * 1024 * 1024 * 1024)
			//sszie := big.NewInt(15 * 1024 * 1024 * 1024 * 1024 * 1024)
			sszieTmp := sszie
			sszieSlice := []big.Int{}
			step := big.Div(sszie, big.NewInt(2880))
			for j := 0; j < 2880; j++ {
				if sszie.GreaterThan(big.Zero()) && j != 2880-1 {
					sszieSlice = append(sszieSlice, step)
					sszie = big.Sub(sszie, step)
				} else {
					sszieSlice = append(sszieSlice, sszie)
					sszie = big.Zero()
				}
			}

			//if i == 0 {
			//	for j := range sszieSlice {
			//		fmt.Println(sszieSlice[j])
			//	}
			//}
			sszie = sszieTmp
			for j := 0; j < 2880; j++ {
				st.updateToNextEpochWithReward(all)
				// only update smoothed estimates after updating reward and epoch
				st.updateSmoothedEstimates(1)
				all = big.Sub(all, sszieSlice[j])
				st1.updateSmoothedEstimate(1)
				circulatingSupply = big.Add(circulatingSupply, big.Mul(big.NewInt(170), big.NewInt(1e18)))
			}
			//fmt.Println(big.Div(st.ThisEpochReward, big.NewInt(5)))
			//fmt.Println(big.Add(st.CumsumRealized, all), st.CumsumBaseline)
			//fmt.Println(st.ThisEpochBaselinePower, all, big.Div(st.ThisEpochReward, big.NewInt(5)))
			//if st.ThisEpochBaselinePower.LessThan(all) {
			//	fmt.Println(i)
			//}

			pledge := InitialPledgeForPower(big.NewInt(32*1024*1024*1024), st.ThisEpochBaselinePower, st.ThisEpochRewardSmoothed,
				st1.ThisEpochQAPowerSmoothed, circulatingSupply)
			st1.ThisEpochRawBytePower = big.Sub(st1.ThisEpochRawBytePower, sszie)
			st1.ThisEpochQualityAdjPower = big.Add(st1.ThisEpochQualityAdjPower, big.Mul(big.NewInt(10), big.NewInt(163840*32*1024*1024*1024)))
			st1.ThisEpochQualityAdjPower = big.Sub(st1.ThisEpochQualityAdjPower, big.NewInt(sint*32*1024*1024*1024))
			cnt1 = append(cnt1, st.ThisEpochBaselinePower)
			cnt2 = append(cnt2, all)
			cnt3 = append(cnt3, big.Div(st.ThisEpochReward, big.NewInt(5)))
			cnt4 = append(cnt4, pledge)
			cnt5 = append(cnt5, st1.ThisEpochQualityAdjPower)
		}
		if true {
			for i := range cnt1 {
				fmt.Printf("%s, %s, %s, %s, %s\n", cnt1[i], cnt2[i], cnt3[i], cnt4[i], cnt5[i])
			}

			//for i := range cnt2 {
			//	fmt.Println(cnt2[i])
			//}
			//
			//fmt.Println()
			//for i := range cnt3 {
			//	fmt.Println(cnt3[i])
			//}
			//
			//fmt.Println()
			//
			//for i := range cnt4 {
			//	fmt.Println(cnt4[i])
			//}
			//fmt.Println()
			//
			//for i := range cnt5 {
			//	fmt.Println(cnt5[i])
			//}
			//fmt.Println()
		}
		//fmt.Println(all)
		return nil
	},
}

var Cal = &cli.Command{
	Name: "cal",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "dsn",
		},
	},
	Action: func(context *cli.Context) error {
		ctx := context.Context
		dsn := context.String("dsn")
		client, err := mongo.NewClient(options.Client().ApplyURI(dsn).SetAppName("bell"))
		if err != nil {
			return fmt.Errorf("new client: %w", err)
		}

		if err := client.Connect(ctx); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		db := client.Database("bell")
		col := db.Collection("MinerSectorHealth")

		cnt0, cnt1 := 0, 0
		m := &MinerInfo{}
		cur, err := col.Find(ctx, bson.M{"Epoch": 2408640 - 2880})
		if err != nil {
			return err
		}
		map2 := map[string]struct{}{}

		for cur.Next(ctx) {
			err = cur.Decode(m)
			if err != nil {
				return err
			}

			a := m.Detail["ActiveSectorsRawPower"].(string)
			sszie, err := strconv.ParseInt(a, 10, 64)
			if err != nil {
				return err
			}

			b := m.Detail["Active"].(int64)
			c := m.Detail["All"].(int64)
			if b == 0 || sszie/b/1024/1024/1024 == 32 {
				cnt1 += 1 * (int)(c)
			} else {
				cnt1 += 2 * (int)(c)
				map2[m.Addr] = struct{}{}
			}
		}
		cur, err = col.Find(ctx, bson.M{"Epoch": 2408640})
		if err != nil {
			return err
		}
		for cur.Next(ctx) {
			err = cur.Decode(m)
			if err != nil {
				return err
			}

			a := m.Detail["ActiveSectorsRawPower"].(string)
			sszie, err := strconv.ParseInt(a, 10, 64)
			if err != nil {
				return err
			}

			b := m.Detail["Active"].(int64)
			c := m.Detail["All"].(int64)
			if _, ok := map2[m.Addr]; ok {
				cnt0 += 2 * (int)(c)
			} else if b == 0 || sszie/b/1024/1024/1024 == 32 {
				cnt0 += 1 * (int)(c)
			} else {
				cnt0 += 2 * (int)(c)
			}
		}

		fmt.Println(cnt0, cnt1)
		fmt.Println(cnt0 - cnt1)
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

func EpochToTime(h abi.ChainEpoch) time.Time {
	return BaseTime.Add(time.Duration(int64(h)*30) * time.Second)
}

//func TimeToEpoch(t time.Time) abi.ChainEpoch {
//	return abi.ChainEpoch(t.Sub(BaseTime).Seconds()) / 30
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

type ExtendConfig struct {
	Extensions []ExtensionConfig
}

type ExtensionConfig struct {
	Sectors       string
	NewExpiration abi.ChainEpoch //可以不填
}

func ParseSectorExtensions(ctx context.Context, path string, maddr address.Address, api v0api.FullNode, ts *types.TipSet, helper bool) (*miner.ExtendSectorExpiration2Params, error) {
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
	var extendConfig ExtendConfig
	err = json.Unmarshal(bytes, &extendConfig)
	if err != nil {
		return nil, err
	}

	extensions := make([]miner.ExpirationExtension2, 0)
	locationMap := make(map[uint64]map[uint64][]uint64)
	seenSectors := make(map[uint64]struct{})
	for _, extension := range extendConfig.Extensions {
		if extension.NewExpiration == 0 && !helper {
			return nil, fmt.Errorf("shoud specify new expiration when helper is false")
		}

		sectors, err := ParseSectorsString(extension.Sectors)
		if err != nil {
			log.Errorf("parse sectors string %v failed: %v", extension.Sectors, err)
			return nil, err
		}

		if len(sectors) == 0 {
			log.Warnf("sectors %v is empty", extension.Sectors)
			continue
		}

		newExpirations := make([]abi.ChainEpoch, 0, len(sectors))
		sectorInfos := make([]*miner.SectorOnChainInfo, 0, len(sectors))
		for _, sector := range sectors {
			if _, ok := seenSectors[sector]; ok {
				log.Warnf("duplicated sector %v", sector)
				continue
			}

			sp, err := api.StateSectorPartition(ctx, maddr, abi.SectorNumber(sector), ts.Key())
			if err != nil {
				log.Errorf("get sector %v location failed: %v", sector, err)
				return nil, err
			}

			if _, ok := locationMap[sp.Deadline]; !ok {
				locationMap[sp.Deadline] = make(map[uint64][]uint64)
				locationMap[sp.Deadline][sp.Partition] = []uint64{sector}
			} else {
				locationMap[sp.Deadline][sp.Partition] = append(locationMap[sp.Deadline][sp.Partition], sector)
			}

			seenSectors[sector] = struct{}{}

			sectorInfo, err := api.StateSectorGetInfo(ctx, maddr, abi.SectorNumber(sector), ts.Key())
			if err != nil {
				log.Errorf("get sector info for sector %v for maddr %v failed: %v", sector, maddr, err)
				return nil, err
			}

			sectorInfos = append(sectorInfos, sectorInfo)
			if helper {
				maxNewExpiration, err := MaxNewExpiration(ts.Height(), sectorInfo)
				if err != nil {
					return nil, err
				}
				newExpirations = append(newExpirations, maxNewExpiration)
			}
		}

		if helper {
			sort.Slice(newExpirations, func(i, j int) bool {
				return newExpirations[i] < newExpirations[j]
			})

			if extension.NewExpiration == 0 {
				extension.NewExpiration = newExpirations[0]
			}
		}

		for _, sectorInfo := range sectorInfos {
			isValidNewExpiration, err := IsValidNewExpiration(ts.Height(), extension.NewExpiration, sectorInfo)
			if !isValidNewExpiration {
				return nil, fmt.Errorf("specified new expiration %v for sector %v is invalid, sectorInfo: %+v", extension.NewExpiration, sectorInfo.SectorNumber, sectorInfo)
			}

			if err != nil {
				return nil, err
			}
		}

		for deadline, location := range locationMap {
			for partition, ss := range location {
				sectorsBitfield := bitfield.NewFromSet(ss)

				extensions = append(extensions, miner.ExpirationExtension2{
					Deadline:      deadline,
					Partition:     partition,
					Sectors:       sectorsBitfield,
					NewExpiration: extension.NewExpiration,
				})
			}
		}
	}

	extendSectorExpiration2Params := &miner.ExtendSectorExpiration2Params{
		Extensions: extensions,
	}
	return extendSectorExpiration2Params, nil
}

// expiration - activation >= miner.MinSectorExpiration
// expiration <= curEpoch + miner.MaxSectorExpirationExtension
// expiration - activation <= maxLifetime
func MaxNewExpiration(curEpoch abi.ChainEpoch, sectorInfo *miner.SectorOnChainInfo) (abi.ChainEpoch, error) {
	maxLifetime, err := builtin.SealProofSectorMaximumLifetime(sectorInfo.SealProof)
	if err != nil {
		return 0, err
	}

	return abi.ChainEpoch(math.Min(float64(curEpoch+miner.MaxSectorExpirationExtension), float64(sectorInfo.Activation+maxLifetime))), nil
}

func IsValidNewExpiration(curEpoch, newExpiration abi.ChainEpoch, sectorInfo *miner.SectorOnChainInfo) (bool, error) {
	maxNewExpiration, err := MaxNewExpiration(curEpoch, sectorInfo)
	if err != nil {
		return false, err
	}

	return newExpiration <= maxNewExpiration && newExpiration-sectorInfo.Activation >= miner.MinSectorExpiration, nil
}

func ParseSectorsString(s string) ([]uint64, error) {
	ss := strings.Split(s, ",")
	res := []uint64{}
	parse := func(sx string) error {
		if strings.Contains(sx, "-") {
			sxs := strings.Split(sx, "-")
			if len(sxs) != 2 {
				return fmt.Errorf("secots string invalid")
			}

			start, err := strconv.ParseUint(strings.TrimSpace(sxs[0]), 10, 64)
			if err != nil {
				return err
			}

			end, err := strconv.ParseUint(strings.TrimSpace(sxs[1]), 10, 64)
			if err != nil {
				return err
			}
			for i := start; i <= end; i++ {
				res = append(res, i)
			}
			return nil
		}

		sx = strings.TrimSpace(sx)
		a, err := strconv.ParseUint(sx, 10, 64)
		if err != nil {
			return fmt.Errorf("secots string invalid %w", err)
		}
		res = append(res, a)
		return nil
	}

	for i := range ss {
		err := parse(ss[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func ParseClaimTerm(ctx context.Context, from address.Address, path string, api v0api.FullNode, helper bool) ([]verifregtypes.ClaimTerm, error) {
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

	for _, claimTerm := range claimTerms {
		// adjust claimTerm is invalid
		provider, err := address.NewIDAddress(uint64(claimTerm.Provider))
		if err != nil {
			return nil, err
		}

		claim, err := api.StateGetClaim(ctx, provider, claimTerm.ClaimId, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		if claim == nil {
			return nil, fmt.Errorf("claim %v not found for provider %v", claimTerm.ClaimId, claimTerm.Provider)
		}

		fromAddr, err := api.StateLookupID(ctx, from, types.EmptyTSK)
		if err != nil {
			return nil, err
		}

		fromID, err := address.IDFromAddress(fromAddr)
		if err != nil {
			return nil, err
		}

		if uint64(claim.Client) != fromID {
			return nil, fmt.Errorf("client %v of claim %v is not equal fromID %v", claim.Client, claimTerm.ClaimId, fromID)
		}

		if helper {
			claimTerm.TermMax = verifregtypes.MaximumVerifiedAllocationTerm
		} else {
			if !IsValidTermMax(*claim, claimTerm.TermMax) {
				return nil, fmt.Errorf("new term_max %v is invalid for claim %v: %+v", claimTerm.TermMax, claimTerm.ClaimId, claim)
			}
		}
	}

	return claimTerms, nil
}

func IsValidTermMax(claim verifregtypes.Claim, newTermMax abi.ChainEpoch) bool {
	return newTermMax <= verifregtypes.MaximumVerifiedAllocationTerm && newTermMax >= claim.TermMax
}

//// deprecated: use ExtendSectorExpiration2 for all
//// IsExtendCommitLegacy return whether to use legacy extend method
//func IsExtendCommitLegacy(ctx context.Context, extendSectorExpiration2Params miner.ExtendSectorExpiration2Params, miner address.Address, api v0api.FullNode, tipset *types.TipSet) (bool, error) {
//	var commitLegacy = true
//	for _, extension := range extendSectorExpiration2Params.Extensions {
//		err := extension.Sectors.ForEach(func(u uint64) error {
//			if commitLegacy == false {
//				return nil
//			}
//
//			sectorInfo, err := api.StateSectorGetInfo(ctx, miner, abi.SectorNumber(u), tipset.Key())
//			if err != nil {
//				return err
//			}
//
//			if sectorInfo.SimpleQAPower && (sectorInfo.VerifiedDealWeight.GreaterThan(big.NewInt(0)) || sectorInfo.DealWeight.GreaterThanEqual(big.NewInt(0))) {
//				commitLegacy = false
//				return nil
//			}
//
//			return nil
//		})
//
//		if err != nil {
//			return commitLegacy, err
//		}
//	}
//
//	return commitLegacy, nil
//}

var errFailedFillSectorsWithClaims = fmt.Errorf("failed fill sectors with claims")

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
			claims, ok := claimsForSectorsMap[abi.SectorNumber(u)]
			if !ok {
				return nil
			}

			maintainClaims, dropClaims := MaintainAndDropClaims(newExpiration, claims)
			if len(dropClaims) > 0 {
				sectorInfo, err := api.StateSectorGetInfo(ctx, provider, abi.SectorNumber(u), tipset.Key())
				if err != nil {
					return err
				}

				canDropClaims := CanDropClaims(sectorInfo.Expiration, tipset.Height())
				if !canDropClaims {
					// new_expiration is too large or extend claims first
					log.Errorf("claims is not allowed to drop because of expiration(%v)-curEpoch(%v) > verifregtypes.EndOfLifeClaimDropPeriod(30d) for sector %v, new expiration %v is too high than claim.term_start + claim.term_max, dropClaims: %v", sectorInfo.Expiration, tipset.Height(), u, newExpiration, dropClaims)
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
func PushMessage(ctx context.Context, api v0api.FullNode, msg *types.Message, confidence uint64) (cid.Cid, error) {
	smsg, err := api.MpoolPushMessage(ctx, msg, nil)
	if err != nil {
		return cid.Undef, err
	}

	// wait for it to get mined into a block
	wait, err := api.StateWaitMsg(ctx, smsg.Cid(), confidence)
	if err != nil {
		return cid.Undef, err
	}

	// check it executed successfully
	if wait.Receipt.ExitCode.IsError() {
		log.Errorf("owner change failed!")
		return cid.Undef, err
	}

	log.Info("message succeeded!")

	return smsg.Cid(), nil
}

type MinerInfo struct {
	Addr   string `bson:"Addr"`
	Detail bson.M `bson:"Detail"`
}

var InitialPledgeMaxPerByte = big.Div(big.NewInt(1e18), big.NewInt(32<<30))

func ExpectedRewardForPower(rewardEstimate, networkQAPowerEstimate smoothing.FilterEstimate,
	qaSectorPower abi.StoragePower, projectionDuration abi.ChainEpoch) abi.TokenAmount {
	networkQAPowerSmoothed := smoothing.Estimate(&networkQAPowerEstimate)
	if networkQAPowerSmoothed.IsZero() {
		return smoothing.Estimate(&rewardEstimate)
	}
	expectedRewardForProvingPeriod := smoothing.ExtrapolatedCumSumOfRatio(projectionDuration, 0, rewardEstimate, networkQAPowerEstimate)
	br128 := big.Mul(qaSectorPower, expectedRewardForProvingPeriod) // Q.0 * Q.128 => Q.128
	br := big.Rsh(br128, smath.Precision128)

	return big.Max(br, big.Zero())
}

func ExpectedRewardForPowerClampedAtAttoFIL(rewardEstimate, networkQAPowerEstimate smoothing.FilterEstimate, qaSectorPower abi.StoragePower, projectionDuration abi.ChainEpoch) abi.TokenAmount {
	br := ExpectedRewardForPower(rewardEstimate, networkQAPowerEstimate, qaSectorPower, projectionDuration)
	if br.LessThanEqual(big.Zero()) {
		br = abi.NewTokenAmount(1)
	}
	return br
}

type BigFrac struct {
	Numerator   big.Int
	Denominator big.Int
}

var InitialPledgeFactor = 20 // PARAM_SPEC
var InitialPledgeProjectionPeriod = abi.ChainEpoch(InitialPledgeFactor) * 2880
var InitialPledgeLockTarget = BigFrac{
	Numerator:   big.NewInt(3), // PARAM_SPEC
	Denominator: big.NewInt(10),
}

func InitialPledgeForPower(qaPower, baselinePower abi.StoragePower, rewardEstimate, networkQAPowerEstimate smoothing.FilterEstimate, circulatingSupply abi.TokenAmount) abi.TokenAmount {
	ipBase := ExpectedRewardForPowerClampedAtAttoFIL(rewardEstimate, networkQAPowerEstimate, qaPower, InitialPledgeProjectionPeriod)

	lockTargetNum := big.Mul(InitialPledgeLockTarget.Numerator, circulatingSupply)
	lockTargetDenom := InitialPledgeLockTarget.Denominator
	pledgeShareNum := qaPower
	networkQAPower := smoothing.Estimate(&networkQAPowerEstimate)
	pledgeShareDenom := big.Max(big.Max(networkQAPower, baselinePower), qaPower) // use qaPower in case others are 0
	additionalIPNum := big.Mul(lockTargetNum, pledgeShareNum)
	additionalIPDenom := big.Mul(lockTargetDenom, pledgeShareDenom)
	additionalIP := big.Div(additionalIPNum, additionalIPDenom)

	nominalPledge := big.Add(ipBase, additionalIP)
	spaceRacePledgeCap := big.Mul(InitialPledgeMaxPerByte, qaPower)
	return big.Min(nominalPledge, spaceRacePledgeCap)
}

func ComputeRTheta(effectiveNetworkTime abi.ChainEpoch, baselinePowerAtEffectiveNetworkTime, cumsumRealized, cumsumBaseline big.Int) big.Int {
	var rewardTheta big.Int
	if effectiveNetworkTime != 0 {
		rewardTheta = big.NewInt(int64(effectiveNetworkTime))  // Q.0
		rewardTheta = big.Lsh(rewardTheta, smath.Precision128) // Q.0 => Q.128
		diff := big.Sub(cumsumBaseline, cumsumRealized)
		diff = big.Lsh(diff, smath.Precision128)                  // Q.0 => Q.128
		diff = big.Div(diff, baselinePowerAtEffectiveNetworkTime) // Q.128 / Q.0 => Q.128
		rewardTheta = big.Sub(rewardTheta, diff)                  // Q.128
	} else {
		// special case for initialization
		rewardTheta = big.Zero()
	}
	return rewardTheta
}

var (
	// lambda = ln(2) / (6 * epochsInYear)
	// for Q.128: int(lambda * 2^128)
	// Calculation here: https://www.wolframalpha.com/input/?i=IntegerPart%5BLog%5B2%5D+%2F+%286+*+%281+year+%2F+30+seconds%29%29+*+2%5E128%5D
	Lambda = big.MustFromString("37396271439864487274534522888786")
	// expLamSubOne = e^lambda - 1
	// for Q.128: int(expLamSubOne * 2^128)
	// Calculation here: https://www.wolframalpha.com/input/?i=IntegerPart%5B%5BExp%5BLog%5B2%5D+%2F+%286+*+%281+year+%2F+30+seconds%29%29%5D+-+1%5D+*+2%5E128%5D
	ExpLamSubOne = big.MustFromString("37396273494747879394193016954629")
)

// Computes a reward for all expected leaders when effective network time changes from prevTheta to currTheta
// Inputs are in Q.128 format
func computeReward(epoch abi.ChainEpoch, prevTheta, currTheta, simpleTotal, baselineTotal big.Int) abi.TokenAmount {
	simpleReward := big.Mul(simpleTotal, ExpLamSubOne)    //Q.0 * Q.128 =>  Q.128
	epochLam := big.Mul(big.NewInt(int64(epoch)), Lambda) // Q.0 * Q.128 => Q.128

	simpleReward = big.Mul(simpleReward, big.NewFromGo(smath.ExpNeg(epochLam.Int))) // Q.128 * Q.128 => Q.256
	simpleReward = big.Rsh(simpleReward, smath.Precision128)                        // Q.256 >> 128 => Q.128

	baselineReward := big.Sub(computeBaselineSupply(currTheta, baselineTotal), computeBaselineSupply(prevTheta, baselineTotal)) // Q.128

	reward := big.Add(simpleReward, baselineReward) // Q.128

	return big.Rsh(reward, smath.Precision128) // Q.128 => Q.0
}

// Computes baseline supply based on theta in Q.128 format.
// Return is in Q.128 format
func computeBaselineSupply(theta, baselineTotal big.Int) big.Int {
	thetaLam := big.Mul(theta, Lambda)               // Q.128 * Q.128 => Q.256
	thetaLam = big.Rsh(thetaLam, smath.Precision128) // Q.256 >> 128 => Q.128

	eTL := big.NewFromGo(smath.ExpNeg(thetaLam.Int)) // Q.128

	one := big.NewInt(1)
	one = big.Lsh(one, smath.Precision128) // Q.0 => Q.128
	oneSub := big.Sub(one, eTL)            // Q.128

	return big.Mul(baselineTotal, oneSub) // Q.0 * Q.128 => Q.128
}

var BaselineExponent = big.MustFromString("340282591298641078465964189926313473653") // Q.128

// 2.5057116798121726 EiB
var BaselineInitialValue = big.NewInt(2_888_888_880_000_000_000) // Q.0

// Initialize baseline power for epoch -1 so that baseline power at epoch 0 is
// BaselineInitialValue.
func InitBaselinePower() abi.StoragePower {
	baselineInitialValue256 := big.Lsh(BaselineInitialValue, 2*smath.Precision128) // Q.0 => Q.256
	baselineAtMinusOne := big.Div(baselineInitialValue256, BaselineExponent)       // Q.256 / Q.128 => Q.128
	return big.Rsh(baselineAtMinusOne, smath.Precision128)                         // Q.128 => Q.0
}

// Compute BaselinePower(t) from BaselinePower(t-1) with an additional multiplication
// of the base exponent.
func BaselinePowerFromPrev(prevEpochBaselinePower abi.StoragePower) abi.StoragePower {
	thisEpochBaselinePower := big.Mul(prevEpochBaselinePower, BaselineExponent) // Q.0 * Q.128 => Q.128
	return big.Rsh(thisEpochBaselinePower, smath.Precision128)                  // Q.128 => Q.0
}

// These numbers are estimates of the onchain constants.  They are good for initializing state in
// devnets and testing but will not match the on chain values exactly which depend on storage onboarding
// and upgrade epoch history. They are in units of attoFIL, 10^-18 FIL
var DefaultSimpleTotal = big.Mul(big.NewInt(330e6), big.NewInt(1e18))   // nolint: varcheck,deadcode
var DefaultBaselineTotal = big.Mul(big.NewInt(770e6), big.NewInt(1e18)) // nolint: varcheck,deadcode
type Spacetime = big.Int

// 36.266260308195979333 FIL
// https://www.wolframalpha.com/input/?i=IntegerPart%5B330%2C000%2C000+*+%281+-+Exp%5B-Log%5B2%5D+%2F+%286+*+%281+year+%2F+30+seconds%29%29%5D%29+*+10%5E18%5D
const InitialRewardPositionEstimateStr = "36266260308195979333"

var InitialRewardPositionEstimate = big.MustFromString(InitialRewardPositionEstimateStr) // nolint: varcheck,deadcode

// -1.0982489*10^-7 FIL per epoch.  Change of simple minted tokens between epochs 0 and 1
// https://www.wolframalpha.com/input/?i=IntegerPart%5B%28Exp%5B-Log%5B2%5D+%2F+%286+*+%281+year+%2F+30+seconds%29%29%5D+-+1%29+*+10%5E18%5D
var InitialRewardVelocityEstimate = abi.NewTokenAmount(-109897758509) // nolint: varcheck,deadcode

type State struct {
	// CumsumBaseline is a target CumsumRealized needs to reach for EffectiveNetworkTime to increase
	// CumsumBaseline and CumsumRealized are expressed in byte-epochs.
	CumsumBaseline Spacetime

	// CumsumRealized is cumulative sum of network power capped by BaselinePower(epoch)
	CumsumRealized Spacetime

	// EffectiveNetworkTime is ceiling of real effective network time `theta` based on
	// CumsumBaselinePower(theta) == CumsumRealizedPower
	// Theta captures the notion of how much the network has progressed in its baseline
	// and in advancing network time.
	EffectiveNetworkTime abi.ChainEpoch

	// EffectiveBaselinePower is the baseline power at the EffectiveNetworkTime epoch
	EffectiveBaselinePower abi.StoragePower

	// The reward to be paid in per WinCount to block producers.
	// The actual reward total paid out depends on the number of winners in any round.
	// This value is recomputed every non-null epoch and used in the next non-null epoch.
	ThisEpochReward abi.TokenAmount
	// Smoothed ThisEpochReward
	ThisEpochRewardSmoothed smoothing.FilterEstimate

	// The baseline power the network is targeting at st.Epoch
	ThisEpochBaselinePower abi.StoragePower

	// Epoch tracks for which epoch the Reward was computed
	Epoch abi.ChainEpoch

	// TotalStoragePowerReward tracks the total FIL awarded to block miners
	TotalStoragePowerReward abi.TokenAmount

	// Simple and Baseline totals are constants used for computing rewards.
	// They are on chain because of a historical fix resetting baseline value
	// in a way that depended on the history leading immediately up to the
	// migration fixing the value.  These values can be moved from state back
	// into a code constant in a subsequent upgrade.
	SimpleTotal   abi.TokenAmount
	BaselineTotal abi.TokenAmount
}

type PowerState struct {
	ThisEpochQualityAdjPower abi.StoragePower
	ThisEpochQAPowerSmoothed smoothing.FilterEstimate
	ThisEpochRawBytePower    abi.StoragePower
}

func (st *PowerState) updateSmoothedEstimate(delta abi.ChainEpoch) {
	filterQAPower := smoothing.LoadFilter(st.ThisEpochQAPowerSmoothed, smoothing.DefaultAlpha, smoothing.DefaultBeta)
	st.ThisEpochQAPowerSmoothed = filterQAPower.NextEstimate(st.ThisEpochQualityAdjPower, delta)
}

//func ConstructState(currRealizedPower abi.StoragePower) *State {
//	st := &State{
//		CumsumBaseline:         big.Zero(),
//		CumsumRealized:         big.Zero(),
//		EffectiveNetworkTime:   0,
//		EffectiveBaselinePower: BaselineInitialValue,
//
//		ThisEpochReward:        big.Zero(),
//		ThisEpochBaselinePower: InitBaselinePower(),
//		Epoch:                  -1,
//
//		ThisEpochRewardSmoothed: smoothing.NewEstimate(InitialRewardPositionEstimate, InitialRewardVelocityEstimate),
//		TotalStoragePowerReward: big.Zero(),
//
//		SimpleTotal:   DefaultSimpleTotal,
//		BaselineTotal: DefaultBaselineTotal,
//	}
//
//	st.updateToNextEpochWithReward(currRealizedPower)
//
//	return st
//}

// Takes in current realized power and updates internal state
// Used for update of internal state during null rounds
func (st *State) updateToNextEpoch(currRealizedPower abi.StoragePower) {
	st.Epoch++
	st.ThisEpochBaselinePower = BaselinePowerFromPrev(st.ThisEpochBaselinePower)
	cappedRealizedPower := big.Min(st.ThisEpochBaselinePower, currRealizedPower)
	st.CumsumRealized = big.Add(st.CumsumRealized, cappedRealizedPower)

	for st.CumsumRealized.GreaterThan(st.CumsumBaseline) {
		st.EffectiveNetworkTime++
		st.EffectiveBaselinePower = BaselinePowerFromPrev(st.EffectiveBaselinePower)
		st.CumsumBaseline = big.Add(st.CumsumBaseline, st.EffectiveBaselinePower)
	}
}

// Takes in a current realized power for a reward epoch and computes
// and updates reward state to track reward for the next epoch
func (st *State) updateToNextEpochWithReward(currRealizedPower abi.StoragePower) {
	prevRewardTheta := ComputeRTheta(st.EffectiveNetworkTime, st.EffectiveBaselinePower, st.CumsumRealized, st.CumsumBaseline)
	st.updateToNextEpoch(currRealizedPower)
	currRewardTheta := ComputeRTheta(st.EffectiveNetworkTime, st.EffectiveBaselinePower, st.CumsumRealized, st.CumsumBaseline)

	st.ThisEpochReward = computeReward(st.Epoch, prevRewardTheta, currRewardTheta, st.SimpleTotal, st.BaselineTotal)
}

func (st *State) updateSmoothedEstimates(delta abi.ChainEpoch) {
	filterReward := smoothing.LoadFilter(st.ThisEpochRewardSmoothed, smoothing.DefaultAlpha, smoothing.DefaultBeta)
	st.ThisEpochRewardSmoothed = filterReward.NextEstimate(st.ThisEpochReward, delta)
}

var bad = `861959
936780
826279
851390
2782784
981155
1000116
1091874
2647688
3125324
1320937
1355826
1407757
1518009
1319357
1276759
1880357
1202528
2572251
2748729
1205108
1112047
2509854
1705796
1299642
1469788
2712838
1394279
1357485
1498835
1448261
1726923
1409724
1650617
1591954
1507353
1489940
1928375
1443784
2703151
1603762
1381291
1628859
1844121
1565489
1294868
1285712
1302605
1517408
1378362
2186529
1525149
1565403
1866104
2539328
2255825
1386080
1445702
1427107
1556045
1455835
1471992
1591598
2371486
1516827
1641159
1591472
2101400
4935473
2903147
2096791
1750434
3313317
1681454
1921936
1760040
3115229
1726920
1899370
1656520
1760002
1926352
1840585
2517212
1640533
2418325
1819535
1601250
1810749
2024168
1701896
1727126
1753645
2036042
1844210
2356221
1668526
1665243
2084174
2582167
1698281
1681983
1722729
1558136
1738388
1918716
1712274
1641692
1741797
1517862
1526783
2191390
1696954
1566021
1808490
1493434
1481406
1378848
1403937
1336890
1435149
1188580
1151190
1705172
1756781
1193451
1148351
1079155
1022015
1193760
3758931
1007027
1103021
1063643
987325
1096000
1097795
907730
1052585
1000241
1376818
1008286
980191
1576592
1078826
1204578
941498
1058732
1150741
1246805
1002244
1052808
1133602
929353
1048735
1090652
1009949
1076287
1171383
1574597
1119433
1406415
900094
1676412
935716
1768644
969977
869846
933110
1050297
888852
888374
998779
943370
917356
988074
931711
957096
2018879
857019
819314
884721
889386
869028
5748356
861170
988751
863270
907804
792528
9881304
886599
701315
1321719
1317941
772757
1122449
762435
752823
1037490
783552
735626
782598
826084
718163
706553
699047
627089
671552
732757
837506
1026554
593169
593366
584333
537492
892985
505023
522061
507951
749323
458705
483213
464712
481767
485399
528593
462110
468232
500739
654499
483152
518764
555498
410695
444970
406523
509563
467095
442394
365816
270931
309015
437984
513090
463321
563895
439316
470107
440089
451493
461916
2710948
422925
429696
423301
418257
442240
458339
632790
453196
539950
475069
622258
481160
474037
521425
611015
517668
490559
578760
531130
565557
618251
601706
489147
448941
454138
775623
490882
482393
552740
863958
524706
538344
1182844
607652
2195555
2296568
518184
486505
737970
484562
490314
517225
526255
719681
577249
571180
610000
1189160
587740
710376
656595
564805
548061
778568
571112
604957
635104
843799
620420
590280
13236468
722135
594412
579676
629320
699634
719601
725936
1460036
900596
514310
1480071
762961
686823
974076
647452
644964
575627
563817
482635
519063
474379
512292
441647
493706
421967
442690
457562
800200
541573
1866158
578418
443914
404174
456148
396657
403495
677422
437111
411486
396019
471813
548920
685924
677987
582768
458406
436571
388805
456155
704087
397212
375371
340186
315683
729392
473017
370323
230438
375854
356406
306296
439067
375625
471422
520093
393339
320567
323769
350648
512504
1138284
313075
465331
457559
279023
262423
279661
311242
329424
299121
671383
1120108
293799
402922
374984
356516
434566
933859
293501
4941020
251938
518764
779871
233000
200306
1110744
253598
953021
425187
212376
244949
660008
198210
238291
199840
265852
300543
714895
392384
466656
599926
786875
874897
316968
325052
599035
241826
220204
406085
508027
269100
236541
528703
256842
329142
580656
647552
650340
549351
252214
267956
338780
352510
317753
250938
265357
255523
430563
389337
200159
464138
355422
243518
275091
272867
229602
186796
407598
1174474
261103
315205
611459
427224
228116
365687
270647
252275
290771
260154
301584
235107
249055
276351
246809
500585
237687
317934
623080
775827
317908
341980
623346
361045
258947
300403
417984
377646
190646
160822
147363
411587
221971
288440
316002
375369
220269
157738
172967
350529
194003
202558
274571
468520
109127
140732
124491
194345
117941
133917
227832
125248
58729
573878
60236
474695
123124
50810
42464
276388
312645
67351
158727
95878
68595
74904
328184
802392
74489
548264
77246
143974
78898
70561
174172
132809
39562
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0
0`
