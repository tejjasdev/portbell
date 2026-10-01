// Package config loads and validates the optional TOML config file.
//
// The file has a version key so that future releases can read old files and
// upgrade them. Unknown keys are errors, which catches typos early.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/tejjasdev/portbell/internal/brand"
	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/responder"
)

// CurrentVersion is the config format version this build reads and writes.
const CurrentVersion = 1

// FileName is the config file looked up in the working directory.
const FileName = brand.ConfigFile

// Config is the full configuration. Every field has a default.
type Config struct {
	Version int    `toml:"version"`
	Server  Server `toml:"server"`
	Reply   Reply  `toml:"reply"`
	Env     Env    `toml:"env"`
	Export  Export `toml:"export"`
}

// Server holds listener settings.
type Server struct {
	Bind         string `toml:"bind"`
	Port         int    `toml:"port"`
	MaxBodyBytes int64  `toml:"max_body_bytes"`
	CORS         bool   `toml:"cors"`
	History      int    `toml:"history"`
}

// Reply is the initial reply sent to every request.
type Reply struct {
	Status  int      `toml:"status"`
	Headers []string `toml:"headers"`
	Body    string   `toml:"body"`
	Delay   string   `toml:"delay"`
}

// Env controls which environment variables ${VAR} placeholders may read.
type Env struct {
	Allow []string `toml:"allow"`
}

// Export controls what leaves the tool through copy and save actions.
type Export struct {
	Redact bool `toml:"redact"`
}

// Defaults returns the built-in configuration.
func Defaults() Config {
	d := responder.Default()
	headers := make([]string, len(d.Headers))
	for i, h := range d.Headers {
		headers[i] = h.Name + ": " + h.Value
	}
	return Config{
		Version: CurrentVersion,
		Server:  Server{Bind: "127.0.0.1", Port: 8080, MaxBodyBytes: 10 << 20, History: events.DefaultCapacity},
		Reply:   Reply{Status: d.Status, Headers: headers, Body: d.Body, Delay: "0s"},
		Export:  Export{Redact: true},
	}
}

// Load reads a config file on top of the defaults.
func Load(path string) (Config, error) {
	cfg := Defaults()
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if un := md.Undecoded(); len(un) > 0 {
		keys := make([]string, len(un))
		for i, k := range un {
			keys[i] = k.String()
		}
		return Config{}, fmt.Errorf("%s: unknown key(s): %s", path, strings.Join(keys, ", "))
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Discover finds the config file to use. An explicit path must exist. Without
// one it tries ./portbell.toml and then the user config directory. found is
// false when no file exists, which is not an error.
func Discover(explicit string) (path string, found bool, err error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", false, fmt.Errorf("config file: %w", err)
		}
		return explicit, true, nil
	}
	candidates := []string{FileName}
	if dir, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(dir, brand.Name, "config.toml"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, true, nil
		}
	}
	return "", false, nil
}

// Validate checks every field.
func (c Config) Validate() error {
	switch {
	case c.Version < 1:
		return errors.New("version must be 1 or higher")
	case c.Version > CurrentVersion:
		return fmt.Errorf("config version %d was written by a newer release; this build reads up to version %d", c.Version, CurrentVersion)
	}
	if strings.TrimSpace(c.Server.Bind) == "" {
		return errors.New("server.bind must not be empty")
	}
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port %d is out of range (0-65535)", c.Server.Port)
	}
	if c.Server.MaxBodyBytes < 1 || c.Server.MaxBodyBytes > 1<<30 {
		return fmt.Errorf("server.max_body_bytes %d is out of range (1-%d)", c.Server.MaxBodyBytes, 1<<30)
	}
	if c.Server.History < 1 || c.Server.History > 100000 {
		return fmt.Errorf("server.history %d is out of range (1-100000)", c.Server.History)
	}
	if _, err := c.Spec(); err != nil {
		return fmt.Errorf("reply: %w", err)
	}
	return nil
}

// Spec converts the [reply] table into a reply Spec.
func (c Config) Spec() (responder.Spec, error) {
	spec := responder.Spec{Status: c.Reply.Status, Body: c.Reply.Body}
	for _, line := range c.Reply.Headers {
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return responder.Spec{}, fmt.Errorf("header %q must look like \"Name: value\"", line)
		}
		spec.Headers = append(spec.Headers, events.Header{Name: strings.TrimSpace(name), Value: strings.TrimSpace(value)})
	}
	if c.Reply.Delay != "" {
		d, err := time.ParseDuration(c.Reply.Delay)
		if err != nil {
			return responder.Spec{}, fmt.Errorf("delay %q is not a duration like 500ms or 2s", c.Reply.Delay)
		}
		spec.Delay = d
	}
	return spec, spec.Validate()
}

// Addr returns the listen address in host:port form.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Server.Bind, strconv.Itoa(c.Server.Port))
}

// IsLoopback reports whether the server is reachable only from this machine.
func (c Config) IsLoopback() bool {
	b := strings.TrimSpace(c.Server.Bind)
	if strings.EqualFold(b, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(b, "[]"))
	return ip != nil && ip.IsLoopback()
}

// EnvPolicy builds the placeholder policy from the [env] table.
func (c Config) EnvPolicy() responder.EnvPolicy {
	return responder.EnvPolicy{Prefix: brand.EnvPrefix, Allow: c.Env.Allow}
}
