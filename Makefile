SHELL = /bin/sh
TARGET = './cmd/pong'
VERSION = $(shell git describe --tags --abbrev=0)
GOFLAGS = -ldflags "-X main.version=$(shell git describe --tags --dirty)"
MODULE_NAME = $(shell go list -m)

install:
	go install $(GOFLAGS) $(TARGET)
.PHONY: install

all:
	@echo ""
	@echo "Version: $(VERSION)"
	@echo ""
	@echo "Semantic executable names use the following syntax:"
	@echo "  <app>-<platform>-<arch>-<version>"
	@echo ""
	@echo ""
	make win32
	make win64
	make darwin32
	make darwin64
	make linux32
	make linux64
.PHONY: all

win32:
	GOOS=windows
	GOARCH=386
	go build $(GOFLAGS) -o bin/pong-win-386-$(VERSION).exe $(TARGET)
.PHONY: win32

win64:
	GOOS=windows
	GOARCH=amd64
	go build $(GOFLAGS) -o bin/pong-win-amd64-$(VERSION).exe $(TARGET)
.PHONY: win64

darwin32:
	GOOS=darwin
	GOARCH=386
	go build $(GOFLAGS) -o bin/pong-darwin-386-$(VERSION) $(TARGET)
.PHONY: darwin32

darwin64:
	GOOS=darwin
	GOARCH=amd64
	go build $(GOFLAGS) -o bin/pong-darwin-amd64-$(VERSION) $(TARGET)
.PHONY: darwin64

linux32:
	GOOS=linux
	GOARCH=386
	go build $(GOFLAGS) -o bin/pong-linux-386-$(VERSION) $(TARGET)
.PHONY: linux32

linux64:
	GOOS=linux
	GOARCH=amd64
	go build $(GOFLAGS) -o bin/pong-linux-amd64-$(VERSION) $(TARGET)
.PHONY: linux64

clean:
	rm -rf ./bin/
.PHONY: clean

run:
	go run ./cmd/pong 192.168.1.1
.PHONY: run

licenses: ## Create Third Party Licenses
	@# requires go-licenses tool
	@# ```shell
	@# go install github.com/google/go-licenses@latest
	@# ```
	@rm -rf licenses
	@go-licenses save ./... --save_path=licenses --force
	@rm -rf licenses/$(MODULE_NAME)
	@find licenses -type d -empty -delete
	@go-licenses csv ./... | grep -v "^$(MODULE_NAME)," > licenses/THIRD_PARTY_LICENSES.csv
.PHONY: licenses
