# Tab'd Native Host

A Go-based native messaging host for the [Tab'd browser extension](https://github.com/iann0036/tabd-extension). This program receives clipboard data from the browser extension and saves it to files in the user's home directory to be picked up by the [IDE extension](https://github.com/iann0036/tabd).

## Installation

Pre-build binaries are packaged with the IDE extension, and can be installed by running the `Tab'd: Install browser helper` command from the Command Palette in your IDE. However, if you want to build it yourself or install it manually, follow the instructions below.

### Quick Install (Recommended)

```bash
# Build and install in one step
./install.sh
```

### Manual Install

```bash
# Build the binary
./build.sh

# Install manually
sudo cp tabd-native-host /usr/local/bin/
chmod +x /usr/local/bin/tabd-native-host

# Install manifest files (see install.sh for details)
```

### Cross-Platform Build

```bash
# Build for all platforms
./build.sh all
```
