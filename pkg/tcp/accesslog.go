package tcp

import (
	"context"
	"github.com/traefik/traefik/v3/pkg/middlewares/accesslogtcp"
)

// AccessLogMiddleware is a TCP middleware that logs connection start and end events.
type AccessLogMiddleware struct {
	handler *accesslogtcp.Handler
	next    Handler
}

// NewAccessLogMiddleware creates a new AccessLogMiddleware.
func NewAccessLogMiddleware(handler *accesslogtcp.Handler) Constructor {
	return func(next Handler) (Handler, error) {
		return &AccessLogMiddleware{
			handler: handler,
			next:    next,
		}, nil
	}
}

// ServeTCP logs connection start and end, then delegates to the next handler.
func (a *AccessLogMiddleware) ServeTCP(conn WriteCloser) {
	ctx := context.Background()
	clientAddr := conn.RemoteAddr().String()
	serverAddr := conn.LocalAddr().String()
	a.handler.LogConnectionStart(ctx, clientAddr, serverAddr)

	// Wrap conn to count bytes and measure duration
	start := nowFunc()
	countingConn := NewCountingConn(conn)
	a.next.ServeTCP(countingConn)
	duration := nowFunc().Sub(start)
	bytesIn, bytesOut := countingConn.BytesRead(), countingConn.BytesWritten()
	a.handler.LogConnectionEnd(ctx, clientAddr, serverAddr, bytesIn, bytesOut, duration, nil)
}

// nowFunc is a variable for testability.
var nowFunc = func() time.Time { return time.Now() }

// CountingConn wraps a WriteCloser to count bytes read/written.
type CountingConn struct {
	WriteCloser
	bytesRead    int64
	bytesWritten int64
}

func NewCountingConn(conn WriteCloser) *CountingConn {
	return &CountingConn{WriteCloser: conn}
}

func (c *CountingConn) Read(b []byte) (int, error) {
	n, err := c.WriteCloser.Read(b)
	c.bytesRead += int64(n)
	return n, err
}

func (c *CountingConn) Write(b []byte) (int, error) {
	n, err := c.WriteCloser.Write(b)
	c.bytesWritten += int64(n)
	return n, err
}

func (c *CountingConn) BytesRead() int64   { return c.bytesRead }
func (c *CountingConn) BytesWritten() int64 { return c.bytesWritten }
