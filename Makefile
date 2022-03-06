#VER = $(shell git describe --tags --abbrev=0)
#DATE = $(shell date "+%Y%m%d%H%M%S")
#HASH = $(shell git rev-parse --short HEAD)
#VERSION = $(VER)-$(DATE)-$(HASH)-mf
VERSION = 0.0.1

pong:
	go build -ldflags '-X "main.version=$(VERSION)"' ./cmd/pong

clean:
	rm -f pong
