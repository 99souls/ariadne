package testsite

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// WithLiveTestSite launches (or reuses) the Bun + React live test site located at
// tools/test-site and calls fn with its base URL.
// Environment variables:
//
//	TESTSITE_PORT   fixed port (optional, else dynamically chosen)
//	TESTSITE_REUSE  set to "1" to reuse an existing instance on TESTSITE_PORT if healthy
//	TESTSITE_ROBOTS robots mode (allow|deny) forwarded to site
func WithLiveTestSite(t *testing.T, fn func(baseURL string)) {
	t.Helper()
	if fn == nil {
		t.Fatalf("nil fn passed to WithLiveTestSite")
	}
	if _, err := exec.LookPath("bun"); err != nil {
		t.Skipf("bun not installed: %v", err)
	}
	repoRoot, err := detectRepoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	reuse := os.Getenv("TESTSITE_REUSE") == "1"
	explicitPort := os.Getenv("TESTSITE_PORT")
	if reuse && explicitPort != "" && healthy("http://127.0.0.1:"+explicitPort) {
		fn("http://127.0.0.1:" + explicitPort)
		return
	}
	port, err := choosePort(explicitPort)
	if err != nil {
		t.Fatalf("choose port: %v", err)
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	cmd := exec.Command("bun", "run", "src/dev.ts")
	cmd.Dir = filepath.Join(repoRoot, "tools", "test-site")
	cmd.Env = append(os.Environ(), fmt.Sprintf("TESTSITE_PORT=%d", port))
	if robots := os.Getenv("TESTSITE_ROBOTS"); robots != "" {
		cmd.Env = append(cmd.Env, "TESTSITE_ROBOTS="+robots)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start test site: %v", err)
	}
	readyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logBuf := &strings.Builder{}
	merged := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(merged)
	readyCh := make(chan struct{})
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			logBuf.WriteString(line + "\n")
			if strings.Contains(line, "TESTSITE: listening on") {
				close(readyCh)
				return
			}
		}
		select {
		case <-readyCh:
		default:
			close(readyCh)
		}
	}()
	select {
	case <-readyCh:
		if !waitHealthy(baseURL, time.Second) {
			_ = cmd.Process.Kill()
			t.Fatalf("health check failed; logs:\n%s", logBuf)
		}
	case <-readyCtx.Done():
		_ = cmd.Process.Kill()
		t.Fatalf("timeout starting test site; partial logs:\n%s", logBuf)
	}
	if !reuse {
		t.Cleanup(func() { shutdown(cmd) })
	}
	fn(baseURL)
}

func shutdown(cmd *exec.Cmd) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = cmd.Process.Signal(os.Interrupt)
	doneCh := make(chan struct{})
	go func() {
		_ = cmd.Wait() // wait for process exit; ignore error since we may interrupt
		close(doneCh)
	}()
	select {
	case <-ctx.Done():
		_ = cmd.Process.Kill()
	case <-doneCh:
	}
}

func choosePort(explicit string) (int, error) {
	if explicit != "" {
		var p int
		_, err := fmt.Sscanf(explicit, "%d", &p)
		if err != nil || p <= 0 {
			return 0, fmt.Errorf("invalid port %q", explicit)
		}
		return p, nil
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = l.Close()
	}()
	if tcp, ok := l.Addr().(*net.TCPAddr); ok {
		return tcp.Port, nil
	}
	// Fallback parse
	_, portStr, perr := net.SplitHostPort(l.Addr().String())
	if perr != nil {
		return 0, perr
	}
	var port int
	_, scanErr := fmt.Sscanf(portStr, "%d", &port)
	if scanErr != nil {
		return 0, scanErr
	}
	return port, nil
}

func healthy(base string) bool { return waitHealthy(base, 200*time.Millisecond) }
func waitHealthy(base string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c := http.Client{Timeout: 200 * time.Millisecond}
		if resp, err := c.Get(base + "/api/ping"); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func detectRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.work")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.work not found")
		}
		dir = parent
	}
}
