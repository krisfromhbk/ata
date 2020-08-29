SOURCE_FILES?=./...

export GO111MODULE := on

build:
	go build cmd/server/main.go
.PHONY: build

test:
	go test -v -failfast -coverprofile=coverage.txt -covermode=atomic $(SOURCE_FILES) -timeout=2m
.PHONY: test

cover: test
	go tool cover -html=coverage.txt
.PHONY: cover