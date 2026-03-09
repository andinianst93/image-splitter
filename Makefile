BINARY   := image-splitter
PKG      := github.com/andinianst93/image-splitter
BIN_DIR  := bin
DIST_DIR := dist
GOOS_LINUX   := linux
GOOS_WINDOWS := windows
GOOS_DARWIN  := darwin

.PHONY: build test run build-all clean

## build: compile the binary for the current platform into bin/
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) .

## test: run all tests
test:
	go test ./...

## run: build and run with a sample invocation (requires INPUT to be set)
##   make run INPUT=photo.jpg ROWS=2 COLS=3
run: build
	./$(BIN_DIR)/$(BINARY) $(INPUT) --rows $(ROWS) --cols $(COLS) --output ./output

## build-all: cross-compile for macOS (arm64/amd64), Linux (amd64), Windows (amd64)
build-all:
	mkdir -p $(DIST_DIR)
	GOOS=$(GOOS_DARWIN)  GOARCH=arm64  go build -o $(DIST_DIR)/$(BINARY)-darwin-arm64 .
	GOOS=$(GOOS_DARWIN)  GOARCH=amd64  go build -o $(DIST_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=$(GOOS_LINUX)   GOARCH=amd64  go build -o $(DIST_DIR)/$(BINARY)-linux-amd64 .
	GOOS=$(GOOS_WINDOWS) GOARCH=amd64  go build -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe .

## clean: remove built binaries and output directory
clean:
	rm -rf $(BIN_DIR)/ $(DIST_DIR)/ output/
