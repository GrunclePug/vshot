# vshot

vshot - An X11-native screen selection and capture tool.

## Overview

`vshot` freezes the screen, lets you select a region with the mouse, and saves it or copies it to the clipboard with vim-style keys. It speaks the X11 protocol directly through the pure-Go [`xgb`](https://github.com/jezek/xgb) library, with no cgo or toolkit dependencies.

## Usage

```bash
vshot          # capture the monitor under the pointer
vshot -all     # capture all monitors
vshot -v       # print version
```

Bind it to a hotkey in your window manager for quick captures.

### Controls

| Input            | Action                                                       |
| ---------------- | ------------------------------------------------------------ |
| Click and drag   | Select a region                                              |
| Drag a handle    | Resize the selection                                         |
| Drag inside      | Move the selection                                           |
| Click outside    | Clear the selection                                          |
| `y`              | Copy the selection (or whole capture) to the clipboard and exit |
| `Esc`            | Clear the selection, or exit if there is none                |
| `:w [file]`      | Save the selection (or whole capture) as PNG                 |
| `:wq [file]`, `:x` | Save, then exit if saving succeeded                        |
| `:q`             | Exit                                                         |

Without a file name, `:w` writes a timestamped file to `DefaultDir`, creating it if needed. Relative names are also placed in `DefaultDir`; absolute paths and `~/` work as expected.

## Installation

### Requirements

* Go 1.22+
* Make
* `xclip` for clipboard support
* An X server with a 24- or 32-bit TrueColor display (practically every modern setup)

### Building

```bash
git clone https://github.com/GrunclePug/vshot.git
cd vshot
make
sudo make install    # installs to /usr/local/bin
```

`make test` runs vet and the unit tests; `make clean` and `sudo make uninstall` undo the above.

### Configuration

Settings are compiled in. Edit `internal/config/config.go` and rebuild:

```go
BorderColor     uint32  = 0xFF00FF
BarColor        uint32  = 0x222222
TextColor       uint32  = 0xbbbbbb
MaskAlpha       float64 = 0.5
DotSize         int     = 4
DefaultDir              = "~/Pictures/Screenshots"
TimestampFormat         = "Screenshot_2006-01-02_150405.png"
```

## Contributing

Contributions are welcome! Fork the repository and submit a pull request.

## Author

Chad Humphries |
[Website](https://grunclepug.com/) |
[GitHub Profile](https://github.com/GrunclePug)

## Other Projects

Check out some of my other projects on GitHub: [Here](https://github.com/GrunclePug?tab=repositories)
