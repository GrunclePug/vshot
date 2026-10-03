# vshot

vshot - A screen selection and capture tool for X11 and Wayland.

## Overview

`vshot` freezes the screen, lets you select a region with the mouse, and saves it or copies it to the clipboard with vim-style keys. One binary works on both X11 and wlroots-based Wayland compositors, speaking each protocol directly with no toolkit.

* **X11:** any window manager (dwm, i3, bspwm, ...), via the pure-Go [`xgb`](https://github.com/jezek/xgb) library.
* **Wayland:** compositors with `wlr-screencopy` and `wlr-layer-shell` (river, sway, Hyprland, ...). GNOME and KDE are not supported.

The backend is picked automatically: Wayland when `WAYLAND_DISPLAY` is set, otherwise X11.

## Usage

```bash
vshot          # capture the monitor under the pointer
vshot -all     # capture all monitors as one image
vshot -backend x11|wayland   # override backend detection
vshot -v       # print version
```

Bind it to a hotkey in your window manager for quick captures.

On Wayland every monitor shows the overlay, and keys act on the monitor under the pointer. With `-all`, selections can span monitors; this requires all monitors to use the same scale. Rotated monitors are not supported yet.

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

* Go 1.22+, a C compiler, `pkg-config` and Make
* `libxkbcommon` (keyboard layouts on Wayland; installed on virtually every Linux desktop)
* Clipboard: `xclip` on X11, `wl-clipboard` (`wl-copy`) on Wayland
* X11: a 24- or 32-bit TrueColor display (practically every modern setup)

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
DotSize         int = 4 // grab handle radius
DefaultDir          = "~/Pictures/Screenshots"
TimestampFormat     = "Screenshot_2006-01-02_150405.png"
```

X11 and Wayland each have their own theme, so vshot can match each desktop:

```go
X11Theme = Theme{
	Border: 0xFF00FF, BorderWidth: 1,
	Bar: 0x222222, Text: 0xbbbbbb,
	Mask: 0x000000, MaskAlpha: 0.5,
}
WaylandTheme = Theme{
	Border: 0x5fd4c4, BorderWidth: 2,
	Bar: 0x0e141a, Text: 0xe6edf0,
	Mask: 0x0e141a, MaskAlpha: 0.5,
}
```

`Mask` is blended over everything outside the selection at strength `MaskAlpha`. Sizes are in logical pixels and scale with HiDPI outputs on Wayland.

## Contributing

Contributions are welcome! Fork the repository and submit a pull request.

## Author

Chad Humphries |
[Website](https://grunclepug.com/) |
[GitHub Profile](https://github.com/GrunclePug)

## Other Projects

Check out some of my other projects on GitHub: [Here](https://github.com/GrunclePug?tab=repositories)
