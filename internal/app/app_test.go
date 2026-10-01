package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tejjasdev/portbell/internal/brand"
	"github.com/tejjasdev/portbell/internal/headless"
)

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func run(args ...string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = Run(context.Background(), args, &o, &e)
	return code, o.String(), e.String()
}

func TestVersionAndHelp(t *testing.T) {
	if code, out, _ := run("--version"); code != ExitOK || !strings.HasPrefix(out, brand.Name+" ") {
		t.Fatalf("version: code=%d out=%q", code, out)
	}
	for _, flag := range []string{"-h", "--help"} {
		code, out, _ := run(flag)
		if code != ExitOK || !strings.Contains(out, "Usage:") || !strings.Contains(out, "--headless") {
			t.Fatalf("%s: code=%d out=%q", flag, code, out)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.toml")
	_ = os.WriteFile(bad, []byte("[server]\nprot = 1\n"), 0o600)

	cases := map[string][]string{
		"unknown flag":       {"--nope"},
		"extra argument":     {"hello"},
		"bad port value":     {"-p", "abc"},
		"port out of range":  {"-p", "70000"},
		"bad config":         {"-c", bad},
		"missing config":     {"-c", filepath.Join(dir, "missing.toml")},
		"missing reply file": {"-r", filepath.Join(dir, "missing.http")},
		"bad status":         {"-s", "42"},
		"negative delay":     {"-d", "-1s"},
		"empty bind":         {"-b", ""},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			code, _, errOut := run(args...)
			if code != ExitUsage {
				t.Fatalf("exit code = %d, want %d (stderr: %s)", code, ExitUsage, errOut)
			}
			if errOut == "" {
				t.Fatal("expected a message on stderr")
			}
		})
	}
}

func TestPortInUseGivesFriendlyMessage(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	code, _, errOut := run("--headless", "-b", "127.0.0.1", "-p", itoa(port))
	if code != ExitRuntime || !strings.Contains(errOut, "already in use") || !strings.Contains(errOut, "-p "+itoa(port+1)) {
		t.Fatalf("code=%d stderr=%q", code, errOut)
	}
}

func TestHeadlessEndToEnd(t *testing.T) {
	dir := t.TempDir()
	reply := filepath.Join(dir, "reply.http")
	_ = os.WriteFile(reply, []byte("HTTP/1.1 201 Created\nContent-Type: application/json\n\n{\"ok\":true}"), 0o600)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr syncBuffer
	done := make(chan int, 1)
	go func() {
		done <- Run(ctx, []string{"--headless", "-b", "127.0.0.1", "-p", "0", "-r", reply, "--cors"}, &stdout, &stderr)
	}()

	re := regexp.MustCompile(`listening on (http://127\.0\.0\.1:\d+)`)
	var base string
	waitFor(t, "startup message", func() bool {
		if m := re.FindStringSubmatch(stderr.String()); m != nil {
			base = m[1]
			return true
		}
		return false
	})

	resp, err := http.Post(base+"/hook?x=1", "application/json", strings.NewReader(`{"n":1}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 || resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("unexpected reply: %d %v", resp.StatusCode, resp.Header)
	}

	waitFor(t, "JSON line", func() bool { return strings.Contains(stdout.String(), "\n") })
	cancel()
	select {
	case code := <-done:
		if code != ExitOK {
			t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after cancel")
	}

	var line headless.Line
	if err := json.Unmarshal([]byte(strings.SplitN(stdout.String(), "\n", 2)[0]), &line); err != nil {
		t.Fatalf("stdout is not JSON lines: %v\n%s", err, stdout.String())
	}
	if line.Schema != headless.Schema || line.Request.Method != "POST" || line.Request.Target != "/hook?x=1" ||
		line.Request.Body.Text != `{"n":1}` || line.Response == nil || line.Response.Status != 201 {
		t.Fatalf("unexpected line: %+v", line)
	}
}

func TestPublicBindPrintsWarning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var stdout, stderr syncBuffer
	done := make(chan int, 1)
	go func() { done <- Run(ctx, []string{"--headless", "-b", "0.0.0.0", "-p", "0"}, &stdout, &stderr) }()
	waitFor(t, "warning", func() bool { return strings.Contains(stderr.String(), "warning:") })
	cancel()
	<-done
	if !strings.Contains(stderr.String(), "anyone on your network") {
		t.Fatalf("stderr: %s", stderr.String())
	}
}

func TestFlagsOverrideConfigFile(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "c.toml")
	_ = os.WriteFile(cfgFile, []byte("[server]\nport = 9100\ncors = true\n[reply]\nstatus = 503\ndelay = \"1s\"\n"), 0o600)

	f := flags{status: 418}
	cfg, spec, err := load(flags{config: cfgFile, status: 418}, map[string]bool{"c": true, "s": true})
	if err != nil {
		t.Fatal(err)
	}
	_ = f
	if cfg.Server.Port != 9100 || !cfg.Server.CORS || spec.Status != 418 || spec.Delay != time.Second {
		t.Fatalf("precedence wrong: port=%d cors=%v status=%d delay=%s", cfg.Server.Port, cfg.Server.CORS, spec.Status, spec.Delay)
	}
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func itoa(n int) string {
	var b [20]byte
	i := len(b)
	for n > 0 || i == len(b) {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
