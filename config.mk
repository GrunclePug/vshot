# vshot version
VERSION = 0.1.0

# paths
PREFIX = /usr/local
MANPREFIX = $(PREFIX)/share/man

# flags
GOFLAGS = 
# -s: disable symbol table
# -w: disable DWARF generation
# -X: inject version string
LDFLAGS = -s -w -X main.Version=${VERSION}

# compiler
GO = go
