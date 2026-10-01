package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tejjasdev/portbell/internal/events"
)

func sample() events.Request {
	return events.Request{
		Method: "POST", Target: "/hook?a=1", Proto: "HTTP/1.1", Host: "localhost:8080",
		Headers: []events.Header{
			{Name: "Authorization", Value: "Bearer secret"},
			{Name: "Content-Length", Value: "9"},
			{Name: "Content-Type", Value: "application/json"},
			{Name: "Host", Value: "localhost:8080"},
		},
		Body: []byte(`{"it's":1}`),
	}
}

func TestCurlRedactsAndQuotes(t *testing.T) {
	got := Curl(sample(), "http", true)
	for _, want := range []string{
		"curl -X 'POST' 'http://localhost:8080/hook?a=1'",
		"-H 'Authorization: REDACTED'",
		"-H 'Content-Type: application/json'",
		`--data-raw '{"it'\''s":1}'`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("curl output missing %q:\n%s", want, got)
		}
	}
	for _, bad := range []string{"secret", "Content-Length", "-H 'Host"} {
		if strings.Contains(got, bad) {
			t.Errorf("curl output must not contain %q:\n%s", bad, got)
		}
	}
	if raw := Curl(sample(), "http", false); !strings.Contains(raw, "Bearer secret") {
		t.Error("redact=false must keep the value")
	}
}

func TestCurlGetWithoutBody(t *testing.T) {
	got := Curl(events.Request{Method: "GET", Target: "/x", Host: "h:1"}, "http", true)
	if got != "curl 'http://h:1/x'" {
		t.Fatalf("got %q", got)
	}
}

func TestCurlOmitsBinaryAndTruncatedBodies(t *testing.T) {
	bin := events.Request{Method: "POST", Target: "/", Host: "h", Body: []byte{0xff, 0x00}}
	if got := Curl(bin, "http", true); strings.Contains(got, "--data-raw") || !strings.Contains(got, "# body omitted") {
		t.Fatalf("binary body must be omitted:\n%s", got)
	}
	cut := events.Request{Method: "POST", Target: "/", Host: "h", Body: []byte("abc"), BodyTruncated: true}
	if got := Curl(cut, "http", true); strings.Contains(got, "--data-raw") {
		t.Fatalf("truncated body must be omitted:\n%s", got)
	}
}

func TestCurlEscapesControlCharactersInURL(t *testing.T) {
	got := Curl(events.Request{Method: "GET", Target: "/\x1b[2J", Host: "h"}, "http", true)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("raw escape byte leaked into curl output: %q", got)
	}
}

func TestRequestFile(t *testing.T) {
	got := RequestFile(sample(), true)
	if !strings.HasPrefix(got, "POST /hook?a=1 HTTP/1.1\n") || !strings.Contains(got, "Authorization: REDACTED") ||
		!strings.HasSuffix(got, `{"it's":1}`) {
		t.Fatalf("unexpected file:\n%s", got)
	}
}

func TestSaveRequestNeverOverwrites(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	ev := events.Event{ID: 7, Request: sample()}
	first, err := SaveRequest(dir, ev, true)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SaveRequest(dir, ev, true)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Base(first) != "request-7-POST.http" || filepath.Base(second) != "request-7-POST-2.http" {
		t.Fatalf("unexpected names: %s, %s", first, second)
	}
	if st, _ := os.Stat(first); st.Mode().Perm()&0o077 != 0 && filepath.Separator == '/' {
		t.Fatalf("file should be owner-only, mode %v", st.Mode())
	}
}

func TestSafeFileName(t *testing.T) {
	for _, ok := range []string{"reply.http", "my reply-1_ok.txt"} {
		if _, err := SafeFileName(ok); err != nil {
			t.Errorf("%q should be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"", ".hidden", "../x", "a/b", `a\b`, "a..b", "x\x00", strings.Repeat("a", 101), "naïve"} {
		if _, err := SafeFileName(bad); err == nil {
			t.Errorf("%q should be refused", bad)
		}
	}
}

func TestCurlHeadUsesDashI(t *testing.T) {
	got := Curl(events.Request{Method: "HEAD", Target: "/x", Host: "h"}, "http", true)
	if got != "curl -I 'http://h/x'" {
		t.Fatalf("got %q", got)
	}
}
