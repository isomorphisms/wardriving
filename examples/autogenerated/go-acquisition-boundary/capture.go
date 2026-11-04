package boundary

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"
)

var ErrOutputLimit = errors.New("diagnostic output limit exceeded")

type CaptureResult struct {
	Stdout, Stderr string
	Elapsed        time.Duration
}
type limitedOutput struct {
	mu     sync.Mutex
	data   []byte
	limit  int
	over   bool
	cancel context.CancelFunc
}

func (w *limitedOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(p)
	remaining := w.limit - len(w.data)
	if n > remaining {
		w.data = append(w.data, p[:remaining]...)
		w.over = true
		w.cancel()
	} else {
		w.data = append(w.data, p...)
	}
	return n, nil
}

// Capture bounds the local command, both streams and inherited-pipe waiting.
// It does not prove a remote rish/Android operation was terminated.
func Capture(parent context.Context, executable string, args []string, limit int) (CaptureResult, error) {
	if limit < 1 || limit > 262144 {
		return CaptureResult{}, errors.New("invalid output budget")
	}
	if _, ok := parent.Deadline(); !ok {
		return CaptureResult{}, errors.New("diagnostic requires deadline")
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	out := &limitedOutput{limit: limit, cancel: cancel}
	errout := &limitedOutput{limit: limit, cancel: cancel}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Stdout = out
	cmd.Stderr = errout
	cmd.WaitDelay = 100 * time.Millisecond
	start := time.Now()
	err := cmd.Run()
	result := CaptureResult{Stdout: string(out.data), Stderr: string(errout.data), Elapsed: time.Since(start)}
	if out.over || errout.over {
		err = ErrOutputLimit
	} else if parent.Err() != nil {
		err = parent.Err()
	}
	return result, err
}
