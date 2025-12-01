BINARY_NAME := draftit
BUILD_DIR := build
WINDOWS_ICON := assets/draft-it-icon.ico
WINDOWS_SYSONAME := draftit_windows.syso

LINUX_AMD64_BIN := $(BUILD_DIR)/linux/amd64/$(BINARY_NAME)
LINUX_ARM64_BIN := $(BUILD_DIR)/linux/arm64/$(BINARY_NAME)
WINDOWS_AMD64_BIN := $(BUILD_DIR)/windows/amd64/$(BINARY_NAME).exe
WINDOWS_ARM64_BIN := $(BUILD_DIR)/windows/arm64/$(BINARY_NAME).exe

CGO_ENABLED ?= 1
LINUX_ARM64_CC ?= aarch64-linux-gnu-gcc
LINUX_ARM64_CXX ?= aarch64-linux-gnu-g++

.PHONY: all build linux windows clean

all: build

build: linux windows

# linux: $(LINUX_AMD64_BIN) $(LINUX_ARM64_BIN)
linux: $(LINUX_AMD64_BIN) 

# windows: $(WINDOWS_AMD64_BIN) $(WINDOWS_ARM64_BIN)
windows: $(WINDOWS_AMD64_BIN) 

.PHONY: check-linux-arm64-cc
check-linux-arm64-cc:
ifeq ($(CGO_ENABLED),1)
	@command -v $(LINUX_ARM64_CC) >/dev/null 2>&1 || { echo "Missing $(LINUX_ARM64_CC) required for linux/arm64 cgo cross-compilation."; exit 1; }
	@command -v $(LINUX_ARM64_CXX) >/dev/null 2>&1 || { echo "Missing $(LINUX_ARM64_CXX) required for linux/arm64 cgo cross-compilation."; exit 1; }
endif

$(WINDOWS_SYSONAME): $(WINDOWS_ICON)
	GO111MODULE=on go run github.com/akavel/rsrc@latest -ico $(WINDOWS_ICON) -o $(WINDOWS_SYSONAME)

$(LINUX_AMD64_BIN):
	@mkdir -p $(dir $@)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 go build -o $@

$(LINUX_ARM64_BIN): | check-linux-arm64-cc
	@mkdir -p $(dir $@)
	CC=$(LINUX_ARM64_CC) CXX=$(LINUX_ARM64_CXX) CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 go build -o $@

$(WINDOWS_AMD64_BIN): $(WINDOWS_SYSONAME)
	@mkdir -p $(dir $@)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 go build -o $@

$(WINDOWS_ARM64_BIN): $(WINDOWS_SYSONAME)
	@mkdir -p $(dir $@)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=arm64 go build -o $@

clean:
	rm -rf $(BUILD_DIR) $(WINDOWS_SYSONAME)
