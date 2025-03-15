# CPU and Memory Monitor

This repository contains two system tray applications written in Go:
- **CPU Monitor**: Displays CPU usage per core and updates a bar chart icon in the system tray.
- **Memory Monitor**: Displays memory usage statistics and updates a pie chart icon in the system tray.

## Prerequisites

Before compiling and running the applications, install the required dependencies:

```sh
# Install Go modules
go mod tidy
```

## Build and Run

### macOS

To build and run the applications on macOS:

```sh
# Build CPU Monitor
go build -o cpu-monitor ./cmd/cpu-monitor
./cpu-monitor

# Build Memory Monitor
go build -o mem-monitor ./cmd/mem-monitor
./mem-monitor
```

### Windows

To build and run the applications on Windows:

```sh
# Build CPU Monitor
go build -ldflags "-s -w -H windowsgui" -o cpu-monitor.exe ./cmd/cpu-monitor
cpu-monitor.exe

# Build Memory Monitor
go build -ldflags "-s -w -H windowsgui" -o mem-monitor.exe ./cmd/mem-monitor
mem-monitor.exe
```

## Dependencies
The following Go modules are used:
- `github.com/Kodeworks/golang-image-ico`: ICO image encoding for system tray icons.
- `github.com/getlantern/systray`: System tray interface for cross-platform support.
- `github.com/shirou/gopsutil`: Fetch system statistics such as CPU and memory usage.

## Features
- Displays live CPU core usage in the system tray.
- Displays memory usage statistics in the system tray.
- Updates system tray icons dynamically to reflect real-time usage.
- Provides an exit option to close the application.

## License
This project is open-source and available under the MIT License.

