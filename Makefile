APP_NAME=edc-pinpad-gateway
BUILD_DIR=../apps
MAIN_FILE=cmd/main.go
LDFLAGS=-s -w

.PHONY: clean build-win build-linux run

# all: clean build-win build-linux

$(BUILD_DIR):
	mkdir $(BUILD_DIR)

build-win: $(BUILD_DIR)
	@echo "🛠️  Building for Windows (amd64)..."
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME).exe $(MAIN_FILE)

build-linux: $(BUILD_DIR)
	@echo "🛠️  Building for Linux (amd64)..."
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

run:
	go run $(MAIN_FILE)

clean:
	rm -rf $(BUILD_DIR)