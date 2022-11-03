FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): build-dep/.filecoin-install ;

MODULES:=

CLEAN:=
BINS:=

ldflags=-X=github.com/filecoin-project/lotus/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	    ldflags+=-extldflags=$(LDFLAGS)
	endif

GOFLAGS+=-ldflags="$(ldflags)"

build-dep/.filecoin-install: $(FFI_PATH)
	    $(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
		    @touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=build-dep/.filecoin-install
CLEAN+=build-dep/.filecoin-install

dev-env:
	ln -s ../../.githooks/pre-push ./.git/hooks/pre-push

$(MODULES): build-dep/.update-modules ;

# dummy file that marks the last time modules were updated
build-dep/.update-modules:
	git submodule update --init --recursive
	touch $@

CLEAN+=build-dep/.update-modules

test: $(BUILD_DEPS)
	go test -v -failfast `go list ./... | grep -v /extern/`

lint: $(BUILD_DEPS)
	golangci-lint run

dep-check: build-dep/.update-modules
	./tool/scripts/submodule-check.sh

build-bell: $(BUILD_DEPS)
	rm -rf ./bell
	go build $(GOFLAGS) -o bell ./cmd/bell

build-bell-grafana: $(BUILD_DEPS)
	rm -rf ./bell-grafana
	go build $(GOFLAGS) -o bell-grafana ./cmd/bell-grafana

build-bell-calib: GOFLAGS+=-tags=calibnet
build-bell-calib: $(BUILD_DEPS)
	rm -rf ./bell
	go build $(GOFLAGS) -o bell ./cmd/bell

build-bell-butterfly: GOFLAGS+=-tags=butterflynet
build-bell-butterfly: $(BUILD_DEPS)
	rm -rf ./bell
	go build $(GOFLAGS) -o bell ./cmd/bell

dist-clean:
	git clean -xdff
	git submodule deinit --all -f


gen-index:
	go run ./tool/genindex/main.go > ./tool/mgoscripts/epoch_indexes.js


gen-model:
	go run ./tool/genschema/main.go > ./tool/analytics/model_schema.md
	go run ./tool/genexamples/main.go > ./tool/analytics/model_example.md

gen-diff:
	go run ./tool/actordiff/main.go > ./tool/upgrade/specs-diffs.txt
	go run ./tool/paramdiff/main.go > ./tool/upgrade/params-n-returns-diffs.txt
	go run ./tool/extractorsdiff/main.go > ./tool/upgrade/extractors-diffs.txt

gen-extractor:
	go run tool/genvercodes/main.go

build-adapter: $(BUILD_DEPS)
	rm -rf ./lotus-api-adapter
	go build $(GOFLAGS) -o lotus-api-adapter ./cmd/londobell-api/adapter

build-adapter-calib: GOFLAGS+=-tags=calibnet
build-adapter-calib: $(BUILD_DEPS)
	rm -rf ./lotus-api-adapter
	go build $(GOFLAGS) -o lotus-api-adapter ./cmd/londobell-api/adapter

build-aggregators: $(BUILD_DEPS)
	rm -rf ./londobell-api-aggregators
	go build $(GOFLAGS) -o londobell-api-aggregators ./cmd/londobell-api/aggregators

build-aggregators-calib: GOFLAGS+=-tags=calibnet
build-aggregators-calib: $(BUILD_DEPS)
	rm -rf ./londobell-api-aggregators
	go build $(GOFLAGS) -o londobell-api-aggregators ./cmd/londobell-api/aggregators

gen-types:
	go run tool/gentypes/main.go
