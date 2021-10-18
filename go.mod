module github.com/dtynn/londobell

go 1.16

require (
	github.com/etclabscore/go-openrpc-reflect v0.0.36 // indirect
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-amt-ipld/v2 v2.1.1-0.20201006184820-924ee87a1349 // indirect
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-paramfetch v0.0.2-0.20210614165157-25a6c7769498 // indirect
	github.com/filecoin-project/go-state-types v0.1.1-0.20210915140513-d354ccf10379
	github.com/filecoin-project/go-statestore v0.1.1 // indirect
	github.com/filecoin-project/lotus v1.12.0
	github.com/filecoin-project/specs-actors v0.9.14
	github.com/filecoin-project/specs-actors/v2 v2.3.5
	github.com/filecoin-project/specs-actors/v3 v3.1.1
	github.com/filecoin-project/specs-actors/v4 v4.0.1
	github.com/filecoin-project/specs-actors/v5 v5.0.4
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/ipld/go-car v0.3.1-0.20210601190600-f512dac51e8e
	github.com/lib/pq v1.7.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multihash v0.0.15
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/prometheus/client_golang v1.10.0
	github.com/raulk/go-watchdog v1.0.1 // indirect
	github.com/robertkrimen/otto v0.0.0-20200922221731-ef014fd054ac
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/supranational/blst v0.1.1 // indirect
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.2.0
	github.com/whyrusleeping/pubsub v0.0.0-20190708150250-92bcb0691325 // indirect
	go.mongodb.org/mongo-driver v1.5.0
	go.uber.org/fx v1.9.0
	go.uber.org/zap v1.16.0
	go4.org v0.0.0-20200411211856-f5505b9728dd
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	launchpad.net/gocheck v0.0.0-20140225173054-000000000087 // indirect
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
