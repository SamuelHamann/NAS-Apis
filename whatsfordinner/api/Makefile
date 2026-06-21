APP_NAME := whatsfordinner
BIN_DIR  := bin

.PHONY: run build test tidy fmt vet clean docker

## run: start the server locally
run:
	go run ./cmd/server

## build: compile the binary into ./bin
build:
	go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

## test: run the test suite
test:
	go test ./...

## tidy: sync go.mod / go.sum
tidy:
	go mod tidy

## fmt: format the code
fmt:
	go fmt ./...

## vet: run static analysis
vet:
	go vet ./...

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## docker: build the container image
docker:
	docker build -t $(APP_NAME):latest .
