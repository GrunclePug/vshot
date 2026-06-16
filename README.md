# vshot

vshot - An X11-native screen selection and capture tool.

## Overview

`vshot` is a lightweight, X11-native tool designed for precise screen region selection and capture. It communicates directly with the X Server, providing a highly responsive experience without the bloat of higher-level windowing toolkits.

## Features

* **X11-Native:** Interfaces directly with the X11 protocol via `xgb` for minimal overhead.
* **Interactive Selection:** Intuitive click-and-drag interface to define capture regions.
* **Persistent State:** Uses a dedicated controller to manage selection state and user interactions smoothly.
* **Modular Architecture:** Clearly separated hardware interaction, UI rendering, and business logic.
* **Minimal:** No heavy dependencies; optimized for performance on X11 desktop environments.

## Usage

You can run the compiled binary directly to start the interactive selection session.

```bash
# Capture a single screen
./bin/vshot

# Capture across all connected displays
./bin/vshot -all

# Check version
./bin/vshot -v
```

### Installation

#### Requirements

* Go 1.22+ (Required for range over int support)
* Make
* Clipboard Utility: `xclip` (preferred) or `xsel` (fallback)
* X11 development headers:
 * Arch Linux : `sudo pacman -S libx11`
 * Debian/Ubuntu: `sudo apt install libx11-dev

#### Configuration

Modify the following options in internal/config/config.go
```go
BorderColor     uint32  = 0xFF00FF
BarColor        uint32  = 0x222222
TextColor       uint32  = 0xbbbbbb
MaskAlpha       float64 = 0.5
DotSize         int     = 4
DefaultDir              = "~/Pictures/Screenshots"
TimestampFormat         = "Screenshot_2006-01-02_150405.png"
```

#### Building

Clone the repository:
`git clone https://github.com/GrunclePug/vshot.git`

Compile the application:
`make`

Install to system:
`sudo make install` (Defaults to /usr/local/bin)

#### Maintenance

Clean build artifacts:
`make clean`

Uninstall from system:
`sudo make uninstall`

## Contributing

Contributions are welcome! If you'd like to contribute to this project, please fork the repository and submit a pull request.

## Author

Chad Humphries |
[Website](https://grunclepug.com/) |
[GitHub Profile](https://github.com/GrunclePug)

## Other Projects

Check out some of my other projects on GitHub: [Here](https://github.com/GrunclePug?tab=repositories)
