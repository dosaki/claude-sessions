BINARY   := claude-sessions-bin
APP_NAME := Claude Sessions.app
PKG      := ./cmd/claude-sessions
VERSION  ?= dev

.PHONY: build test app build-windows icon clean

## build: compile the binary for the current platform
build:
	go build -o $(BINARY) $(PKG)

## test: run all tests
test:
	go test ./...

## app: build a macOS .app bundle (run on macOS only)
app: build icon
	rm -rf "$(APP_NAME)"
	mkdir -p "$(APP_NAME)/Contents/MacOS"
	mkdir -p "$(APP_NAME)/Contents/Resources"
	@printf '%s\n' \
		'<?xml version="1.0" encoding="UTF-8"?>' \
		'<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"' \
		'  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' \
		'<plist version="1.0">' \
		'<dict>' \
		'  <key>CFBundleName</key>' \
		'  <string>Claude Sessions</string>' \
		'  <key>CFBundleDisplayName</key>' \
		'  <string>Claude Sessions</string>' \
		'  <key>CFBundleIdentifier</key>' \
		'  <string>com.claude-sessions.app</string>' \
		'  <key>CFBundleVersion</key>' \
		'  <string>$(VERSION)</string>' \
		'  <key>CFBundleShortVersionString</key>' \
		'  <string>$(VERSION)</string>' \
		'  <key>CFBundleExecutable</key>' \
		'  <string>claude-sessions</string>' \
		'  <key>CFBundleIconFile</key>' \
		'  <string>icon</string>' \
		'  <key>CFBundlePackageType</key>' \
		'  <string>APPL</string>' \
		'  <key>NSHighResolutionCapable</key>' \
		'  <true/>' \
		'</dict>' \
		'</plist>' > "$(APP_NAME)/Contents/Info.plist"
	cp $(BINARY) "$(APP_NAME)/Contents/MacOS/claude-sessions"
	@if [ -f assets/icon.icns ]; then \
		cp assets/icon.icns "$(APP_NAME)/Contents/Resources/icon.icns"; \
	fi
	@echo "Built $(APP_NAME) (version $(VERSION))"

## build-windows: cross-compile for Windows with -H windowsgui
build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o claude-sessions.exe $(PKG)

## icon: generate macOS .icns from docs/logo.svg (requires rsvg-convert or sips)
icon:
	@mkdir -p assets
	@mkdir -p icon.iconset
	@# Convert SVG to 1024x1024 PNG
	@if command -v rsvg-convert >/dev/null 2>&1; then \
		rsvg-convert -w 1024 -h 1024 docs/logo.svg -o icon_1024.png; \
	elif command -v sips >/dev/null 2>&1; then \
		qlmanage -t -s 1024 -o . docs/logo.svg 2>/dev/null && mv logo.svg.png icon_1024.png; \
	else \
		echo "Error: need rsvg-convert or sips to convert SVG"; exit 1; \
	fi
	@# Generate iconset sizes
	@for size in 16 32 64 128 256 512; do \
		sips -z $$size $$size icon_1024.png --out icon.iconset/icon_$${size}x$${size}.png >/dev/null 2>&1; \
	done
	@sips -z 32 32 icon_1024.png --out icon.iconset/icon_16x16@2x.png >/dev/null 2>&1
	@sips -z 64 64 icon_1024.png --out icon.iconset/icon_32x32@2x.png >/dev/null 2>&1
	@sips -z 256 256 icon_1024.png --out icon.iconset/icon_128x128@2x.png >/dev/null 2>&1
	@sips -z 512 512 icon_1024.png --out icon.iconset/icon_256x256@2x.png >/dev/null 2>&1
	@sips -z 1024 1024 icon_1024.png --out icon.iconset/icon_512x512@2x.png >/dev/null 2>&1
	iconutil -c icns icon.iconset -o assets/icon.icns
	@rm -rf icon.iconset icon_1024.png
	@echo "Generated assets/icon.icns"

## clean: remove build artifacts
clean:
	rm -f $(BINARY) claude-sessions.exe
	rm -rf "$(APP_NAME)"
	rm -rf assets/
