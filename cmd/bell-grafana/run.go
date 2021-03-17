package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipld/go-car"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"

	// register schemas
	_ "github.com/dtynn/londobell/racailum/segment/extract/actorstate"
	_ "github.com/dtynn/londobell/racailum/segment/extract/tipset"
	_ "github.com/dtynn/londobell/racailum/segment/model"

	"github.com/dtynn/londobell/lib/grafana"
)

var runCmd = &cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name: "listen",
		},
	},
	Action: func(cctx *cli.Context) error {
		addr := "0.0.0.0:15502"
		if lis := cctx.String("listen"); lis != "" {
			addr = lis
		}

		genBytes := build.MaybeGenesis()
		cr, err := car.NewCarReader(bytes.NewBuffer(genBytes))
		if err != nil {
			return err
		}

		var gblk blocks.Block

	BLK_LOOP:
		for {
			blk, err := cr.Next()
			switch err {
			case io.EOF:
				break BLK_LOOP

			case nil:
				if blk.Cid() == cr.Header.Roots[0] {
					gblk = blk
					break BLK_LOOP
				}

			default:
				return err
			}
		}

		if gblk == nil {
			return fmt.Errorf("genesis root blk %s not found", cr.Header.Roots[0])
		}

		gbh, err := types.DecodeBlock(gblk.RawData())
		if err != nil {
			return err
		}

		gr, err := grafana.New(gbh)
		if err != nil {
			return err
		}

		log.Infow("listen", "addr", addr)

		return http.ListenAndServe(addr, cors(gr))
	},
}

func cors(inner http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Access-Control-Allow-Headers", "accept, content-type")
		rw.Header().Set("Access-Control-Allow-Methods", "*")
		rw.Header().Set("Access-Control-Allow-Origin", "*")

		inner.ServeHTTP(rw, r)
	}
}
