package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "c.toml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultsAreValid(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadOverridesOnlyGivenKeys(t *testing.T) {
	p := write(t, `
version = 1
[server]
port = 9000
cors = true
[reply]
status = 503
headers = ["Retry-After: 5"]
delay = "250ms"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9000 || !cfg.Server.CORS || cfg.Server.Bind != "127.0.0.1" {
		t.Fatalf("unexpected server config: %+v", cfg.Server)
	}
	spec, err := cfg.Spec()
	if err != nil {
		t.Fatal(err)
	}
	if spec.Status != 503 || spec.Delay != 250*time.Millisecond || spec.Headers[0].Name != "Retry-After" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
	if cfg.Addr() != "127.0.0.1:9000" || !cfg.IsLoopback() {
		t.Fatalf("addr/loopback wrong: %s %v", cfg.Addr(), cfg.IsLoopback())
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	cases := map[string]string{
		"unknown key":      "[server]\nprot = 1\n",
		"newer version":    "version = 2\n",
		"port range":       "[server]\nport = 70000\n",
		"bad delay":        "[reply]\ndelay = \"soon\"\n",
		"bad header":       "[reply]\nheaders = [\"nocolon\"]\n",
		"bad status":       "[reply]\nstatus = 42\n",
		"not toml":         "this is not toml",
		"empty bind":       "[server]\nbind = \"\"\n",
		"history too high": "[server]\nhistory = 99999999\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(write(t, content)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestUnknownKeyMessageNamesTheKey(t *testing.T) {
	_, err := Load(write(t, "[server]\nprot = 1\n"))
	if err == nil || !strings.Contains(err.Error(), "server.prot") {
		t.Fatalf("error should name the key, got %v", err)
	}
}

func TestIsLoopback(t *testing.T) {
	for bind, want := range map[string]bool{
		"127.0.0.1": true, "localhost": true, "::1": true, "[::1]": true,
		"0.0.0.0": false, "192.168.1.5": false, "::": false,
	} {
		c := Defaults()
		c.Server.Bind = bind
		if got := c.IsLoopback(); got != want {
			t.Errorf("IsLoopback(%q) = %v, want %v", bind, got, want)
		}
	}
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "none"))
	t.Setenv("HOME", filepath.Join(dir, "none"))
	t.Setenv("AppData", filepath.Join(dir, "none"))

	if _, found, err := Discover(""); found || err != nil {
		t.Fatalf("expected nothing found, got found=%v err=%v", found, err)
	}
	if err := os.WriteFile(FileName, []byte("version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if p, found, _ := Discover(""); !found || p != FileName {
		t.Fatalf("expected %s, got %q %v", FileName, p, found)
	}
	if _, _, err := Discover("missing.toml"); err == nil {
		t.Fatal("an explicit missing path must be an error")
	}
}
