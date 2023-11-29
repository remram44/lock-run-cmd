Version := $(shell git describe --tags --dirty)
GitCommit := $(shell git rev-parse HEAD)
LDFLAGS := "-s -w -X main.Version=$(Version) -X main.GitCommit=$(GitCommit)"
NAME := lockrun

.PHONY: all
all: dist

.PHONY: test
test:
	go test -v ./...

.PHONY: dist linux-amd64 darwin linux-armv7 linux-arm64 windows-amd64
dist: linux-amd64 darwin linux-armv7 linux-arm64 windows-amd64

linux-amd64:
	mkdir -p bin/
	GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -ldflags $(LDFLAGS) -o bin/$(NAME)-amd64 cmd/lockrun/lockrun.go

darwin:
	mkdir -p bin/
	GOARCH=amd64 CGO_ENABLED=0 GOOS=darwin go build -ldflags $(LDFLAGS) -o bin/$(NAME)-darwin cmd/lockrun/lockrun.go

linux-armv7:
	mkdir -p bin/
	GOARM=7 GOARCH=arm CGO_ENABLED=0 GOOS=linux go build -ldflags $(LDFLAGS) -o bin/$(NAME)-arm cmd/lockrun/lockrun.go

linux-arm64:
	mkdir -p bin/
	GOARCH=arm64 CGO_ENABLED=0 GOOS=linux go build -ldflags $(LDFLAGS) -o bin/$(NAME)-arm64 cmd/lockrun/lockrun.go

windows-amd64:
	mkdir -p bin/
	GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o bin/$(NAME).exe cmd/lockrun/lockrun.go
