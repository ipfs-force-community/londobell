#FFI_PATH:=../../extern/filecoin-ffi/     #??
#FFI_DEPS:=.install-filcrypto
#FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_deps))
#
#build-dep/.filecoin-install: $(FFI_PATH)
#	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
#		@touch $@

ldflags=-X=github.com/filecoin-project/lotus/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"
GOFLAGS+=-tags=replay

build-replay:
	rm -rf ./replay
	go build $(GOFLAGS) -o replay ./cmd/replay
	go run github.com/GeertJohan/go.rice/rice append --exec replay -i ./build


#BUILD_DEPS+=build-dep/.filecoin-install

#link-build-dir:
#	./tool/scripts/link-build.sh
#BUILD_DEPS+=link-build-dir


#CLEAN:=
#
#CLEAN+=build-dep/.filecoin-install
#
#build-dep/.update-modules:
#	git submodule update --init --recursive
#	touch $@
#
#CLEAN+=build-dep/.update-modules

dist-clean:
	git clean -xdff
	git submodule deinit --all -f