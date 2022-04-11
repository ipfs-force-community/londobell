package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"

	"github.com/ipfs-force-community/londobell/cmd/lotus-api-adapter/controller"
)

var testCmd = &cli.Command{
	Name: "test",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "tipset",
		},
	},
	Action: func(cctx *cli.Context) error {
		err := controller.GetFullNodeAPI(cctx.Context)
		if err != nil {
			return err
		}

		tsk, err := parsetTipSetKey(cctx.String("tipset"))
		if err != nil {
			return err
		}

		cids := tsk.Cids()
		b, err := controller.API.ChainGetBlock(cctx.Context, cids[0])
		if err != nil {
			return err
		}

		// new a block
		var b2 types.BlockHeader
		b2.Miner = b.Miner
		b2.Ticket = b.Ticket
		b2.ElectionProof = b.ElectionProof
		b2.BeaconEntries = b.BeaconEntries
		b2.WinPoStProof = b.WinPoStProof
		b2.Parents = b.Parents
		b2.ParentWeight = b.ParentWeight
		b2.Height = b.Height
		b2.ParentStateRoot = b.ParentStateRoot
		b2.ParentMessageReceipts = b.ParentMessageReceipts
		b2.Messages = b.Messages
		b2.BLSAggregate = b.BLSAggregate
		b2.Timestamp = b.Timestamp + 1
		b2.BlockSig = b.BlockSig
		b.ForkSignaling = b.ForkSignaling
		b.ParentBaseFee = b.ParentBaseFee

		fmt.Println(cids[0])
		fmt.Println(b2.Cid())

		return nil
	},
}

func parsetTipSetKey(s string) (types.TipSetKey, error) {
	cids, err := lcli.ParseTipSetString(s)
	if err != nil {
		return types.EmptyTSK, err
	}

	return types.NewTipSetKey(cids...), nil
}
