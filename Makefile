BIN := $(HOME)/.local/bin

.PHONY: build test lint install

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

# В ~/.local/bin, а не через go install: GOBIN у mise смотрит внутрь его
# каталога, а ProxyCommand ищет devproxy в PATH неинтерактивного шелла.
install:
	go build -o $(BIN)/devc ./cmd/devc
	ln -sfn devc $(BIN)/devproxy
