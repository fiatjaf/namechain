all: bin/bmm_generate bin/bmm_mine bin/named bin/namecli

bin/named: $(shell find ./named -name "*.go")
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/named github.com/fiatjaf/namechain/named

bin/namecli: $(shell find ./cli -name "*.go")
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/namecli github.com/fiatjaf/namechain/cli

bin/bmm_generate: $(shell find ./bmm/generate -name "*.go")
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/bmm_generate github.com/fiatjaf/namechain/bmm/generate

bin/bmm_mine: $(shell find ./bmm/mine -name "*.go")
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/bmm_mine github.com/fiatjaf/namechain/bmm/mine
