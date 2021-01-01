# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gfaf android ios gfaf-cross swarm evm all test clean
.PHONY: gfaf-linux gfaf-linux-386 gfaf-linux-amd64 gfaf-linux-mips64 gfaf-linux-mips64le
.PHONY: gfaf-linux-arm gfaf-linux-arm-5 gfaf-linux-arm-6 gfaf-linux-arm-7 gfaf-linux-arm64
.PHONY: gfaf-darwin gfaf-darwin-386 gfaf-darwin-amd64
.PHONY: gfaf-windows gfaf-windows-386 gfaf-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gfaf:
	build/env.sh go run build/ci.go install ./cmd/gfaf
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gfaf\" to launch gfaf."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gfaf.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gfaf.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	./build/clean_go_build_cache.sh
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

swarm-devtools:
	env GOBIN= go install ./cmd/swarm/mimegen

# Cross Compilation Targets (xgo)

gfaf-cross: gfaf-linux gfaf-darwin gfaf-windows gfaf-android gfaf-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-*

gfaf-linux: gfaf-linux-386 gfaf-linux-amd64 gfaf-linux-arm gfaf-linux-mips64 gfaf-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-*

gfaf-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gfaf
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep 386

gfaf-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gfaf
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep amd64

gfaf-linux-arm: gfaf-linux-arm-5 gfaf-linux-arm-6 gfaf-linux-arm-7 gfaf-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep arm

gfaf-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gfaf
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep arm-5

gfaf-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gfaf
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep arm-6

gfaf-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gfaf
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep arm-7

gfaf-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gfaf
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep arm64

gfaf-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gfaf
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep mips

gfaf-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gfaf
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep mipsle

gfaf-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gfaf
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep mips64

gfaf-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gfaf
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-linux-* | grep mips64le

gfaf-darwin: gfaf-darwin-386 gfaf-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-darwin-*

gfaf-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gfaf
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-darwin-* | grep 386

gfaf-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gfaf
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-darwin-* | grep amd64

gfaf-windows: gfaf-windows-386 gfaf-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-windows-*

gfaf-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gfaf
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-windows-* | grep 386

gfaf-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gfaf
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gfaf-windows-* | grep amd64
