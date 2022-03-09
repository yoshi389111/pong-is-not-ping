SHELL = /bin/bash
TARGET = './src'
VERSION = $(shell git describe --tags --abbrev=0)
GOFLAGS = -ldflags "-X main.version=$(VERSION)"

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

win32:
	GOOS=windows
	GOARCH=386
	go build $(GOFLAGS) -o bin/pong-win-amd64-$(VERSION).exe $(TARGET)
win64:
	GOOS=windows
	GOARCH=amd64
	go build $(GOFLAGS) -o bin/pong-win-386-$(VERSION).exe $(TARGET)

darwin32:
	GOOS=darwin
	GOARCH=386
	go build $(GOFLAGS) -o bin/pong-darwin-386-$(VERSION) $(TARGET)

darwin64:
	GOOS=darwin
	GOARCH=amd64
	go build $(GOFLAGS) -o bin/pong-darwin-amd64-$(VERSION) $(TARGET)

linux32:
	GOOS=linux
	GOARCH=386
	go build $(GOFLAGS) -o bin/pong-linux-386-$(VERSION) $(TARGET)

linux64:
	GOOS=linux
	GOARCH=amd64
	go build $(GOFLAGS) -o bin/pong-linux-amd64-$(VERSION) $(TARGET)

clean:
	printf "IGNORE ANY THROWN ERRORS...\n"
	rm -f ./bin/
	del ./bin/ -Force
