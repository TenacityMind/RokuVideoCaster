BINARY_NAME=roku-caster
VERSION=2.0.0

.PHONY: all build-mac build-win build-all package-channel clean run

all: build-mac

# Package Roku SceneGraph Channel
package-channel:
	@echo "Packaging Roku SceneGraph channel..."
	@cd internal/roku/channel && zip -r -9 ../channel.zip manifest source components images
	@echo "Channel packaged to internal/roku/channel.zip"

# Build macOS binary (Native)
build-mac: package-channel
	@echo "Building macOS standalone binary..."
	@mkdir -p bin
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/roku-caster
	@echo "Done: bin/$(BINARY_NAME)"

# Build Windows 64-bit executable (.exe)
build-win: package-channel
	@echo "Building Windows standalone executable..."
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/$(BINARY_NAME)-win.exe ./cmd/roku-caster
	@echo "Done: bin/$(BINARY_NAME)-win.exe"

# Cross-compile for all platforms
build-all: build-mac build-win
	@echo "All binaries built in bin/"

# Run locally
run: build-mac
	./bin/$(BINARY_NAME)

clean:
	rm -rf bin/
