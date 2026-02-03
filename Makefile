.PHONY: build run test clean lint dev build-all

BINARY=quality-bot
BUILD_DIR=./bin
SRC_DIR=./cmd/quality-bot

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(SRC_DIR)

run: build
	$(BUILD_DIR)/$(BINARY) $(ARGS)

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f *.log

lint:
	golangci-lint run ./...

# Development
dev:
	go run $(SRC_DIR) $(ARGS)

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(SRC_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe $(SRC_DIR)

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run analysis on a repo
analyze:
	$(BUILD_DIR)/$(BINARY) analyze --repo $(REPO)
