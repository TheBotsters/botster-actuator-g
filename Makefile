BINARY   := actuator-g
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)

.PHONY: build build-linux build-darwin build-all test clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/actuator-g

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-linux-amd64 ./cmd/actuator-g

build-darwin:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-arm64 ./cmd/actuator-g

build-all: build-linux build-darwin

test:
	go test ./...

clean:
	rm -f $(BINARY) $(BINARY)-*
