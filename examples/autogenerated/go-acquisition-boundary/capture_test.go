package boundary

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestCaptureHelper(t *testing.T) {
	if len(os.Args) < 3 {
		return
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "flood":
		for i := 0; i < 65536; i++ {
			io.WriteString(os.Stdout, "012345678901234567890123456789\n")
		}
		os.Exit(0)
	case "stderr":
		for i := 0; i < 65536; i++ {
			io.WriteString(os.Stderr, "012345678901234567890123456789\n")
		}
		os.Exit(0)
	case "hang":
		time.Sleep(10 * time.Second)
		os.Exit(0)
	case "ok":
		io.WriteString(os.Stdout, "ok")
		os.Exit(0)
	}
}

func TestBoundedCapture(t *testing.T) {
	for _, mode := range []string{"ok", "flood", "stderr", "hang"} {
		t.Run(mode, func(t *testing.T) {
			timeout := 300 * time.Millisecond
			if mode == "ok" {
				timeout = 3 * time.Second
			} // race runtime has an exit grace period
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			r, err := Capture(ctx, os.Args[0], []string{"-test.run=^TestCaptureHelper$", "--", mode}, 256)
			if len(r.Stdout) > 256 || len(r.Stderr) > 256 || r.Elapsed > 4*time.Second {
				t.Fatalf("unbounded diagnostic: %+v %v", r, err)
			}
			switch mode {
			case "ok":
				if err != nil || r.Stdout != "ok" {
					t.Fatalf("%+v %v", r, err)
				}
			case "hang":
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
			default:
				if !errors.Is(err, ErrOutputLimit) {
					t.Fatal(err)
				}
			}
		})
	}
}
