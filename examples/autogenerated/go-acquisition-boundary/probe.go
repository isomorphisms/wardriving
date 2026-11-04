// Package boundary is an executable design experiment, not the Android adapter.
package boundary

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrBinding   = errors.New("selected Wi-Fi Network and bound resolver/dialer required")
	ErrStale     = errors.New("connection generation replaced")
	ErrBodyLimit = errors.New("response body exceeds observation limit")
	ErrEndpoint  = errors.New("unexpected endpoint response; internet status unresolved")
)

type Network struct {
	Handle, Generation uint64
	WiFi               bool
}

// BoundDialer MUST use network-specific resolution AND socket binding. A
// ControlContext socket hook with default-network DNS does not implement it.
type BoundDialer func(context.Context, string, string) (net.Conn, error)
type DialerFactory func(Network) (BoundDialer, error)

type Config struct {
	Endpoint       string
	ExpectedStatus int
	Timeout        time.Duration
	MaxBody        int64
	// TLS is nil in production; tests install their loopback server CA.
	TLS *tls.Config
	// DisableReuse exists only to measure the forced-close comparison.
	DisableReuse bool
}

type counters struct{ sent, received atomic.Int64 }
type countedConn struct {
	net.Conn
	totals *counters
}

func (c *countedConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	c.totals.received.Add(int64(n))
	return n, e
}
func (c *countedConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	c.totals.sent.Add(int64(n))
	return n, e
}

type generation struct {
	network   Network
	ctx       context.Context
	cancel    context.CancelFunc
	transport *http.Transport
	client    *http.Client
	totals    counters
}

type Engine struct {
	mu             sync.Mutex
	config         Config
	factory        DialerFactory
	current        *generation
	lastGeneration uint64
	// Probes are coalesced in the production engine; the experiment admits
	// only one so cumulative socket counters have an explicit boundary.
	probing bool
}

type Observation struct {
	Network                    Network
	Status                     int
	Redirect                   string
	Elapsed                    time.Duration
	Reused                     bool
	TLSResumed                 bool
	BodyBytes                  int64
	SocketSent, SocketReceived int64
}

func New(config Config, factory DialerFactory) (*Engine, error) {
	u, err := url.Parse(config.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" ||
		factory == nil || config.Timeout <= 0 || config.MaxBody < 0 || config.MaxBody > 65536 ||
		config.ExpectedStatus < 200 || config.ExpectedStatus >= 300 {
		return nil, errors.New("invalid bounded HTTPS endpoint contract")
	}
	if config.TLS != nil {
		if config.TLS.InsecureSkipVerify {
			return nil, errors.New("TLS verification cannot be disabled")
		}
		config.TLS = config.TLS.Clone()
	}
	return &Engine{config: config, factory: factory}, nil
}

func (e *Engine) SetNetwork(network Network) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Delayed callbacks cannot retire a newer connection.
	if network.Generation <= e.lastGeneration {
		return ErrStale
	}
	e.lastGeneration = network.Generation
	// Retire the previous pool even when the replacement is unusable.
	if e.current != nil {
		e.current.cancel()
		e.current.transport.CloseIdleConnections()
		e.current = nil
	}
	if network.Handle == 0 || !network.WiFi {
		return ErrBinding
	}
	dial, err := e.factory(network)
	if err != nil || dial == nil {
		return ErrBinding
	}
	g := &generation{network: network}
	g.ctx, g.cancel = context.WithCancel(context.Background())
	g.transport = &http.Transport{
		Proxy: nil, // Never silently inherit the build host's proxy path.
		DialContext: func(ctx context.Context, kind, address string) (net.Conn, error) {
			c, err := dial(ctx, kind, address)
			if err != nil {
				return nil, err
			}
			return &countedConn{Conn: c, totals: &g.totals}, nil
		},
		TLSClientConfig:        e.config.TLS,
		TLSHandshakeTimeout:    e.config.Timeout,
		ResponseHeaderTimeout:  e.config.Timeout,
		MaxResponseHeaderBytes: 8192,
		MaxConnsPerHost:        1, MaxIdleConnsPerHost: 1,
		IdleConnTimeout:    30 * time.Second,
		DisableKeepAlives:  e.config.DisableReuse,
		DisableCompression: true,
	}
	g.client = &http.Client{Transport: g.transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	e.current = g
	return nil
}

func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.current != nil {
		e.current.cancel()
		e.current.transport.CloseIdleConnections()
		e.current = nil
	}
}

func (e *Engine) Probe(parent context.Context) (out Observation, err error) {
	e.mu.Lock()
	g := e.current
	if g == nil {
		e.mu.Unlock()
		return out, ErrBinding
	}
	if e.probing {
		e.mu.Unlock()
		return out, errors.New("probe already running")
	}
	e.probing = true
	e.mu.Unlock()
	defer func() { e.mu.Lock(); e.probing = false; e.mu.Unlock() }()
	started := time.Now()
	out.Network = g.network
	ctx, cancel := context.WithTimeout(parent, e.config.Timeout)
	stop := context.AfterFunc(g.ctx, cancel)
	defer func() { stop(); cancel(); out.Elapsed = time.Since(started) }()
	if g.ctx.Err() != nil {
		return out, ErrStale
	}
	trace := &httptrace.ClientTrace{
		GotConn:          func(info httptrace.GotConnInfo) { out.Reused = info.Reused },
		TLSHandshakeDone: func(state tls.ConnectionState, _ error) { out.TLSResumed = state.DidResume },
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, trace), http.MethodGet, e.config.Endpoint, nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("User-Agent", "Wardriving-boundary/1")
	resp, err := g.client.Do(req)
	if err == nil {
		out.Status = resp.StatusCode
		out.Redirect = resp.Header.Get("Location")
		// Read at most budget+1 even if a portal sends an unbounded page.
		var n int64
		n, err = io.Copy(io.Discard, io.LimitReader(resp.Body, e.config.MaxBody+1))
		out.BodyBytes = n
		resp.Body.Close()
		if n > e.config.MaxBody {
			err = ErrBodyLimit
		}
		if err == nil && out.Status != e.config.ExpectedStatus {
			err = ErrEndpoint
		}
	}
	out.SocketSent = g.totals.sent.Load()
	out.SocketReceived = g.totals.received.Load()
	// These counters are cumulative per-generation, not per-request wire
	// bytes. TLS post-handshake traffic may be read asynchronously.
	e.mu.Lock()
	stale := e.current != g || g.ctx.Err() != nil
	e.mu.Unlock()
	if stale {
		return out, ErrStale
	}
	return out, err
}
