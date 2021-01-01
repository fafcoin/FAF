.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/fafereum/go-fafereum/cmd/swarm
	go build -o bin/gfaf     github.com/fafereum/go-fafereum/cmd/gfaf
	go build -o bin/bootnode github.com/fafereum/go-fafereum/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/fafereum/go-fafereum/swarm/...
