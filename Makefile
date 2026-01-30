.PHONY: server plugin clean install help

# Default target
help:
	@echo "Available targets:"
	@echo "  make server  - Run the local bridge server"
	@echo "  make plugin  - Package Bob plugin as .bobplugin"
	@echo "  make install - Package and install plugin to Bob"
	@echo "  make clean   - Remove built artifacts"

# Run the Go server
server:
	cd server && go run .

# Build server binary
build:
	cd server && go build -o ../bin/bedrock-bridge .

# Package Bob plugin
plugin:
	@rm -f bob-plugin-bedrock.bobplugin
	zip -j bob-plugin-bedrock.bobplugin info.json main.js
	@echo "Created bob-plugin-bedrock.bobplugin"

# Package and install plugin to Bob
install: plugin
	open bob-plugin-bedrock.bobplugin

# Clean built artifacts
clean:
	rm -f bob-plugin-bedrock.bobplugin
	rm -rf bin/
