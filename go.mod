module github.com/dtynn/londobell

go 1.16

require (
	github.com/etclabscore/go-openrpc-reflect v0.0.36 // indirect
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-commp-utils v0.1.0 // indirect
	github.com/filecoin-project/go-fil-markets v1.2.5 // indirect
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-paramfetch v0.0.2-0.20210614165157-25a6c7769498 // indirect
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/go-statestore v0.1.1 // indirect
	github.com/filecoin-project/lotus v1.2.0
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v2 v2.3.5-0.20210114162132-5b58b773f4fb
	github.com/filecoin-project/specs-actors/v3 v3.1.0
	github.com/filecoin-project/specs-actors/v4 v4.0.0
	github.com/filecoin-project/specs-actors/v5 v5.0.1
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/ipfs/go-metrics-interface v0.0.1
	github.com/ipld/go-car v0.1.1-0.20201119040415-11b6074b6d4d
	github.com/libp2p/go-libp2p-pubsub v0.4.2-0.20210212194758-6c1addf493eb // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multihash v0.0.14
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/onsi/gomega v1.10.5 // indirect
	github.com/prometheus/client_golang v1.6.0
	github.com/raulk/go-watchdog v1.0.1 // indirect
	github.com/robertkrimen/otto v0.0.0-20200922221731-ef014fd054ac
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/syndtr/goleveldb v1.0.0
	github.com/urfave/cli/v2 v2.2.0
	github.com/whyrusleeping/pubsub v0.0.0-20190708150250-92bcb0691325 // indirect
	go.mongodb.org/mongo-driver v1.5.0
	go.opencensus.io v0.22.5 // indirect
	go.uber.org/fx v1.9.0
	go.uber.org/zap v1.16.0
	go4.org v0.0.0-20200411211856-f5505b9728dd
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	honnef.co/go/tools v0.0.1-2020.1.3 // indirect
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
