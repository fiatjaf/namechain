all: bin/named bin/namecli

bin/named: $(shell find ./named -name "*.go")
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./named github.com/fiatjaf/namechain/named

bin/namecli: $(shell find ./namecli -name "*.go")
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o ./namecli github.com/fiatjaf/namechain/namecli
