# vshot - An X11-native screen selection and capture tool.
# See LICENSE file for copyright and license details.

include config.mk




all: vshot

vshot:
	@mkdir -p bin
	${GO} build ${GOFLAGS} -ldflags "${LDFLAGS}" -o bin/vshot ./cmd/vshot

clean:
	rm -rf bin

install: vshot
	@mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp -f bin/vshot $(DESTDIR)$(PREFIX)/bin
	@chmod 755 $(DESTDIR)$(PREFIX)/bin/vshot

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/vshot

.PHONY: all clean install uninstall

