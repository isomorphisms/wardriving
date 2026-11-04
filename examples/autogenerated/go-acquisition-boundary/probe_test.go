package boundary

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fixture(t testing.TB, handler http.HandlerFunc, reuse bool) (*Engine, *atomic.Int64) {
	t.Helper()
	s := httptest.NewUnstartedServer(handler)
	s.Config.ErrorLog = log.New(io.Discard, "", 0)
	s.StartTLS()
	t.Cleanup(s.Close)
	roots := x509.NewCertPool()
	roots.AddCert(s.Certificate())
	dials := &atomic.Int64{}
	factory := func(Network) (BoundDialer, error) {
		return func(ctx context.Context, kind, address string) (net.Conn, error) {
			dials.Add(1)
			return (&net.Dialer{}).DialContext(ctx, kind, address)
		}, nil
	}
	e, err := New(Config{Endpoint: s.URL, ExpectedStatus: 204, Timeout: time.Second, MaxBody: 1024, TLS: &tls.Config{RootCAs: roots, ClientSessionCache: tls.NewLRUClientSessionCache(4)}, DisableReuse: !reuse}, factory)
	if err != nil {
		t.Fatal(err)
	}
	if err = e.SetNetwork(Network{Handle: 42, Generation: 1, WiFi: true}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.Close)
	return e, dials
}

func TestPoolReuseAndNetworkReplacement(t *testing.T) {
	e, dials := fixture(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }, true)
	for i := 0; i < 3; i++ {
		o, err := e.Probe(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if o.Reused != (i > 0) {
			t.Fatalf("reuse[%d]=%v", i, o.Reused)
		}
	}
	if dials.Load() != 1 {
		t.Fatalf("wanted one TCP connection, got %d", dials.Load())
	}
	if err := e.SetNetwork(Network{Handle: 43, Generation: 2, WiFi: true}); err != nil {
		t.Fatal(err)
	}
	o, err := e.Probe(context.Background())
	if err != nil || o.Reused || dials.Load() != 2 {
		t.Fatalf("new generation reused old pool: %+v %v", o, err)
	}
	if err = e.SetNetwork(Network{Handle: 42, Generation: 1, WiFi: true}); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	o, err = e.Probe(context.Background())
	if err != nil || o.Network.Generation != 2 {
		t.Fatalf("stale callback changed current state: %+v %v", o, err)
	}
}

func TestForcedCloseAndTLSResumption(t *testing.T) {
	e, dials := fixture(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }, false)
	first, err := e.Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if dials.Load() != 2 || first.TLSResumed || !second.TLSResumed || second.Reused {
		t.Fatalf("unexpected resumption: first=%+v second=%+v dials=%d", first, second, dials.Load())
	}
}

func TestRedirectObservedWithoutFollowing(t *testing.T) {
	var hits atomic.Int64
	e, _ := fixture(t, func(w http.ResponseWriter, r *http.Request) { hits.Add(1); http.Redirect(w, r, "/login", 302) }, true)
	o, err := e.Probe(context.Background())
	if !errors.Is(err, ErrEndpoint) || o.Status != 302 || o.Redirect != "/login" || hits.Load() != 1 {
		t.Fatalf("redirect escaped observation: %+v %v hits=%d", o, err, hits.Load())
	}
}

func TestPortalBodyBound(t *testing.T) {
	e, _ := fixture(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, strings.Repeat("x", 65536)) }, true)
	o, err := e.Probe(context.Background())
	if !errors.Is(err, ErrBodyLimit) || o.BodyBytes != 1025 {
		t.Fatalf("unbounded body: %+v %v", o, err)
	}
}

func TestDeadlineAfterResponseHeaders(t *testing.T) {
	e, _ := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}, true)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	o, err := e.Probe(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || o.Elapsed > time.Second {
		t.Fatalf("read did not cancel: %+v %v", o, err)
	}
}

func TestGenerationCancelsInflightAndRejectsLateResult(t *testing.T) {
	entered := make(chan struct{})
	e, _ := fixture(t, func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }, true)
	done := make(chan error, 1)
	go func() { _, err := e.Probe(context.Background()); done <- err }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler not reached")
	}
	if err := e.SetNetwork(Network{Handle: 43, Generation: 2, WiFi: true}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, ErrStale) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("obsolete probe not cancelled")
	}
}

func TestWrongTransportAndMissingBindingProduceNoTraffic(t *testing.T) {
	var calls atomic.Int64
	e, err := New(Config{Endpoint: "https://example.invalid/", ExpectedStatus: 204, Timeout: time.Second}, func(Network) (BoundDialer, error) { calls.Add(1); return nil, ErrBinding })
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	for _, n := range []Network{{Handle: 1, Generation: 1, WiFi: false}, {Handle: 0, Generation: 2, WiFi: true}, {Handle: 2, Generation: 3, WiFi: true}} {
		if err = e.SetNetwork(n); !errors.Is(err, ErrBinding) {
			t.Fatal(err)
		}
		if _, err = e.Probe(context.Background()); !errors.Is(err, ErrBinding) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("ineligible transport reached dialer factory: %d", calls.Load())
	}
}

func TestTLSVerificationRemainsMandatory(t *testing.T) {
	e, _ := fixture(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }, true)
	e.config.TLS = nil // use system trust for the untrusted loopback certificate
	if err := e.SetNetwork(Network{Handle: 43, Generation: 2, WiFi: true}); err != nil {
		t.Fatal(err)
	}
	if o, err := e.Probe(context.Background()); err == nil || o.Status != 0 {
		t.Fatalf("untrusted TLS passed: %+v %v", o, err)
	}
	_, err := New(Config{Endpoint: "https://example.invalid", ExpectedStatus: 204, Timeout: time.Second, TLS: &tls.Config{InsecureSkipVerify: true}}, func(Network) (BoundDialer, error) { return nil, nil })
	if err == nil {
		t.Fatal("insecure TLS configuration admitted")
	}
}

func BenchmarkTLSProbe(b *testing.B) {
	for _, reuse := range []bool{false, true} {
		b.Run(fmt.Sprintf("reuse=%v", reuse), func(b *testing.B) {
			e, dials := fixture(b, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }, reuse)
			b.ResetTimer()
			var last Observation
			for i := 0; i < b.N; i++ {
				var err error
				last, err = e.Probe(context.Background())
				if err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(dials.Load())/float64(b.N), "connections/op")
			b.ReportMetric(float64(last.SocketSent+last.SocketReceived)/float64(b.N), "socket-bytes/op")
		})
	}
}
