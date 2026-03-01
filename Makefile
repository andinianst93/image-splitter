BINARY   := image-splitter
PKG      := github.com/andinianst93/image-splitter
GOOS_LINUX   := linux
GOOS_WINDOWS := windows
GOOS_DARWIN  := darwin

.PHONY: build test run build-all clean

## build: compile the binary for the current platform
build:
	go build -o $(BINARY) .

## test: run all tests
test:
	go test ./...

## run: build and run with a sample invocation (requires INPUT to be set)
##   make run INPUT=photo.jpg ROWS=2 COLS=3
run: build
	./$(BINARY) $(INPUT) --rows $(ROWS) --cols $(COLS) --output ./output

## build-all: cross-compile for macOS (arm64/amd64), Linux (amd64), Windows (amd64)
build-all:
	GOOS=$(GOOS_DARWIN)  GOARCH=arm64  go build -o dist/$(BINARY)-darwin-arm64 .
	GOOS=$(GOOS_DARWIN)  GOARCH=amd64  go build -o dist/$(BINARY)-darwin-amd64 .
	GOOS=$(GOOS_LINUX)   GOARCH=amd64  go build -o dist/$(BINARY)-linux-amd64 .
	GOOS=$(GOOS_WINDOWS) GOARCH=amd64  go build -o dist/$(BINARY)-windows-amd64.exe .

## clean: remove built binaries and output directory
clean:
	rm -f $(BINARY) $(BINARY).exe
	rm -rf dist/ output/
