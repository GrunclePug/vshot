package wayland

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// conn is a minimal Wayland client connection: it encodes requests,
// decodes events and routes them to per-object handlers. Only the argument
// types vshot uses are supported.
type conn struct {
	sock     *net.UnixConn
	handlers map[uint32]handler
	nextID   uint32
	in       []byte
	fds      []int
	rbuf     []byte
	err      error // fatal protocol error reported by the compositor
}

type handler func(op uint16, m *msg)

// fd marks a request argument as a file descriptor to pass.
type fd int

const displayID = 1

func dial() (*conn, error) {
	name := os.Getenv("WAYLAND_DISPLAY")
	if name == "" {
		name = "wayland-0"
	}
	if !filepath.IsAbs(name) {
		dir := os.Getenv("XDG_RUNTIME_DIR")
		if dir == "" {
			return nil, errors.New("XDG_RUNTIME_DIR is not set")
		}
		name = filepath.Join(dir, name)
	}
	sock, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: name, Net: "unix"})
	if err != nil {
		return nil, err
	}

	c := &conn{sock: sock, handlers: map[uint32]handler{}, nextID: displayID + 1}
	c.handlers[displayID] = func(op uint16, m *msg) {
		switch op {
		case 0: // error
			obj, code, text := m.uint(), m.uint(), m.string()
			c.err = fmt.Errorf("wayland protocol error on object %d (code %d): %s", obj, code, text)
		case 1: // delete_id
			delete(c.handlers, m.uint())
		}
	}
	return c, nil
}

func (c *conn) close() { c.sock.Close() }

// newID allocates an object id whose events go to h (which may be nil).
func (c *conn) newID(h handler) uint32 {
	id := c.nextID
	c.nextID++
	c.handlers[id] = h
	return id
}

// send writes request op on object id. Arguments may be uint32, int32,
// string or fd.
func (c *conn) send(id uint32, op uint16, args ...any) error {
	buf := make([]byte, 8, 64)
	var fds []int
	for _, a := range args {
		switch v := a.(type) {
		case uint32:
			buf = binary.LittleEndian.AppendUint32(buf, v)
		case int32:
			buf = binary.LittleEndian.AppendUint32(buf, uint32(v))
		case string:
			n := len(v) + 1
			buf = binary.LittleEndian.AppendUint32(buf, uint32(n))
			buf = append(buf, v...)
			buf = append(buf, make([]byte, pad(n)-len(v))...)
		case fd:
			fds = append(fds, int(v))
		default:
			panic(fmt.Sprintf("wayland: unsupported argument type %T", a))
		}
	}
	binary.LittleEndian.PutUint32(buf[0:], id)
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(buf))<<16|uint32(op))

	var oob []byte
	if len(fds) > 0 {
		oob = syscall.UnixRights(fds...)
	}
	_, _, err := c.sock.WriteMsgUnix(buf, oob, nil)
	return err
}

// dispatch blocks until data arrives, then handles every complete event.
func (c *conn) dispatch() error {
	if c.rbuf == nil {
		c.rbuf = make([]byte, 4096+syscall.CmsgSpace(28*4))
	}
	buf, oob := c.rbuf[:4096], c.rbuf[4096:]
	n, oobn, _, _, err := c.sock.ReadMsgUnix(buf, oob)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("wayland compositor closed the connection")
	}
	c.in = append(c.in, buf[:n]...)
	if oobn > 0 {
		msgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			return err
		}
		for _, m := range msgs {
			fds, err := syscall.ParseUnixRights(&m)
			if err != nil {
				return err
			}
			c.fds = append(c.fds, fds...)
		}
	}

	for len(c.in) >= 8 {
		id := binary.LittleEndian.Uint32(c.in)
		word := binary.LittleEndian.Uint32(c.in[4:])
		size := int(word >> 16)
		if size < 8 {
			return errors.New("wayland: malformed event header")
		}
		if len(c.in) < size {
			break
		}
		if h := c.handlers[id]; h != nil {
			h(uint16(word), &msg{c: c, data: c.in[8:size]})
		}
		c.in = c.in[size:]
		if c.err != nil {
			return c.err
		}
	}
	return nil
}

// roundtrip waits until the compositor has processed every request sent
// so far, handling events in the meantime.
func (c *conn) roundtrip() error {
	done := false
	cb := c.newID(func(uint16, *msg) { done = true })
	if err := c.send(displayID, 0, cb); err != nil { // wl_display.sync
		return err
	}
	for !done {
		if err := c.dispatch(); err != nil {
			return err
		}
	}
	return nil
}

func pad(n int) int { return (n + 3) &^ 3 }

// msg decodes the arguments of one event in order.
type msg struct {
	c    *conn
	data []byte
}

func (m *msg) uint() uint32 {
	if len(m.data) < 4 {
		return 0
	}
	v := binary.LittleEndian.Uint32(m.data)
	m.data = m.data[4:]
	return v
}

func (m *msg) int() int32 { return int32(m.uint()) }

func (m *msg) fixed() float64 { return float64(m.int()) / 256 }

func (m *msg) string() string {
	b := m.array()
	if len(b) > 0 {
		b = b[:len(b)-1] // trailing NUL
	}
	return string(b)
}

func (m *msg) array() []byte {
	n := int(m.uint())
	if n > len(m.data) {
		n = len(m.data)
	}
	b := m.data[:n]
	m.data = m.data[min(pad(n), len(m.data)):]
	return b
}

// fd takes the next file descriptor received with the event stream.
func (m *msg) fd() int {
	if len(m.c.fds) == 0 {
		return -1
	}
	f := m.c.fds[0]
	m.c.fds = m.c.fds[1:]
	return f
}
