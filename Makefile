.PHONY: all build build-py build-go install-py install-go

PY_ENTRY := bin/py800mon
GO_BIN := bin/go800mon

all: build

build: build-py build-go

build-go:
	go build -C go800mon -o ../$(GO_BIN) ./cmd/go800mon

install-py:
	pip install --user .

install-go: build-go
	go -C go800mon install ./cmd/go800mon
