# Makefile for allmend

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
APP_NAME = allmend
SRC_DIR = ./cmd/allmend

.PHONY: all build install clean test fmt lint

all: build

build:
	go build -o $(APP_NAME) $(SRC_DIR)

fmt:
	go fmt ./...

lint:
	go vet ./...

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(APP_NAME) $(DESTDIR)$(BINDIR)/$(APP_NAME)

clean:
	rm -f $(APP_NAME)

test:
	go test ./...
