// Package export turns captured requests into things a person can copy or
// save: a curl command, or a plain HTTP message file.
//
// Secrets are hidden by default. Authorization, Cookie and similar headers are
// replaced with REDACTED unless the caller asks otherwise.
package export

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/safe"
)

// Redacted replaces the value of a sensitive header.
const Redacted = "REDACTED"

// DefaultRedactHeaders lists headers whose values are hidden on export.
var DefaultRedactHeaders = []string{"Authorization", "Cookie", "Proxy-Authorization", "X-Api-Key", "Set-Cookie"}

// skippedInCurl are headers curl sets by itself.
var skippedInCurl = map[string]bool{"host": true, "content-length": true, "transfer-encoding": true, "connection": true}

// Curl returns a curl command (POSIX shell syntax) that repeats the request.
func Curl(req events.Request, scheme string, redact bool) string {
	if scheme == "" {
		scheme = "http"
	}
	var parts []string
	first := "curl"
	switch {
	case req.Method == "HEAD":
		first += " -I" // curl -X HEAD would hang waiting for a body
	case req.Method != "" && req.Method != "GET":
		first += " -X " + shellQuote(safe.Line(req.Method))
	}
	parts = append(parts, first+" "+shellQuote(safe.Line(scheme+"://"+req.Host+req.Target)))
	for _, h := range req.Headers {
		if skippedInCurl[strings.ToLower(h.Name)] {
			continue
		}
		parts = append(parts, "-H "+shellQuote(safe.Line(h.Name+": "+headerValue(h, redact))))
	}
	note := ""
	switch {
	case len(req.Body) == 0:
	case req.BodyTruncated || !textBody(req.Body):
		note = "\n# body omitted: it is binary, contains control characters, or was cut at the size limit"
	default:
		parts = append(parts, "--data-raw "+shellQuote(string(req.Body)))
	}
	return strings.Join(parts, " \\\n  ") + note
}

// RequestFile returns the request as a plain HTTP message.
func RequestFile(req events.Request, redact bool) string {
	var b strings.Builder
	proto := req.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	fmt.Fprintf(&b, "%s %s %s\n", safe.Line(req.Method), safe.Line(req.Target), safe.Line(proto))
	for _, h := range req.Headers {
		fmt.Fprintf(&b, "%s: %s\n", safe.Line(h.Name), safe.Line(headerValue(h, redact)))
	}
	b.WriteString("\n")
	switch {
	case len(req.Body) == 0:
	case textBody(req.Body):
		b.Write(req.Body)
		if req.BodyTruncated {
			b.WriteString("\n# body cut at the size limit")
		}
	default:
		fmt.Fprintf(&b, "# binary body, %d bytes, not saved", len(req.Body))
	}
	return b.String()
}

// SaveRequest writes the request into dir and returns the file path. It never
// overwrites an existing file; it picks the next free name instead. The file is
// readable only by its owner because requests can contain secrets.
func SaveRequest(dir string, ev events.Event, redact bool) (string, error) {
	if dir == "" {
		dir = "."
	}
	// Create directory with 0o700 (owner only) to protect saved requests containing sensitive data.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	method := cleanName(ev.Request.Method)
	if method == "" {
		method = "request"
	}
	content := []byte(RequestFile(ev.Request, redact))
	for n := 0; n < 1000; n++ {
		name := fmt.Sprintf("request-%d-%s.http", ev.ID, method)
		if n > 0 {
			name = fmt.Sprintf("request-%d-%s-%d.http", ev.ID, method, n+1)
		}
		path := filepath.Join(dir, name)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(content); err != nil {
			_ = f.Close()
			return "", err
		}
		return path, f.Close()
	}
	return "", errors.New("could not find a free file name")
}

// SafeFileName checks a file name chosen by a person. It refuses anything that
// could leave the target folder or hide itself.
func SafeFileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errors.New("file name is empty")
	case len(name) > 100:
		return "", errors.New("file name is longer than 100 characters")
	case strings.HasPrefix(name, "."):
		return "", errors.New("file name must not start with a dot")
	case strings.Contains(name, ".."):
		return "", errors.New("file name must not contain \"..\"")
	}
	for _, r := range name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '.' || r == '_' || r == '-' || r == ' '
		if !ok {
			return "", fmt.Errorf("file name contains %q; use letters, digits, space, dot, dash or underscore", r)
		}
	}
	return name, nil
}

func headerValue(h events.Header, redact bool) string {
	if !redact {
		return h.Value
	}
	for _, name := range DefaultRedactHeaders {
		if strings.EqualFold(h.Name, name) {
			return Redacted
		}
	}
	return h.Value
}

func textBody(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	return safe.Text(string(b)) == string(b)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func cleanName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
