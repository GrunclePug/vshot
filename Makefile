# vshot - A screen selection and capture tool for X11 and Wayland.
# See LICENSE file for copyright and license details.

include config.mk

SRC = $(shell find cmd internal -name '*.go') go.mod go.sum config.mk

all: bin/vshot

# A real file target, so `sudo make install` after `make` only copies and
# doesn't rebuild as root.
bin/vshot: ${SRC}
	@mkdir -p bin
	${GO} build ${GOFLAGS} -ldflags "${LDFLAGS}" -o $@ ./cmd/vshot

test:
	${GO} vet ./...
	${GO} test ./...

clean:
	rm -rf bin

install: bin/vshot
	@mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp -f bin/vshot $(DESTDIR)$(PREFIX)/bin
	@chmod 755 $(DESTDIR)$(PREFIX)/bin/vshot

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/vshot

.PHONY: all test clean install uninstall
