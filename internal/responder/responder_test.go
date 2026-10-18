package responder

import (
	"strings"
	"testing"
	"time"

	"github.com/tejjasdev/portbell/internal/events"
)

func TestValidate(t *testing.T) {
	good := Default()
	if err := good.Validate(); err != nil {
		t.Fatalf("default spec invalid: %v", err)
	}
	bad := []struct {
		name string
		spec Spec
	}{
		{"status low", Spec{Status: 99}},
		{"status informational", Spec{Status: 199}},
		{"status high", Spec{Status: 600}},
		{"delay negative", Spec{Status: 200, Delay: -1}},
		{"delay too long", Spec{Status: 200, Delay: time.Hour}},
		{"header name empty", Spec{Status: 200, Headers: []events.Header{{Name: "", Value: "x"}}}},
		{"header name space", Spec{Status: 200, Headers: []events.Header{{Name: "Bad Name", Value: "x"}}}},
		{"header value newline", Spec{Status: 200, Headers: []events.Header{{Name: "X", Value: "a\r\nInjected: 1"}}}},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.spec.Validate(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestEnvExpandOnlyAllowedNames(t *testing.T) {
	env := map[string]string{"PORTBELL_TOKEN": "abc", "SECRET_KEY": "s3cr3t", "EXTRA": "ok"}
	p := EnvPolicy{
		Prefix: "PORTBELL_",
		Allow:  []string{"EXTRA"},
		Lookup: func(k string) (string, bool) { v, ok := env[k]; return v, ok },
	}
	got := p.Expand("t=${PORTBELL_TOKEN} s=${SECRET_KEY} e=${EXTRA} m=${PORTBELL_MISSING}")
	want := "t=abc s=${SECRET_KEY} e=ok m=${PORTBELL_MISSING}"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResponderSetValidatesAndRenders(t *testing.T) {
	r := New(Default(), EnvPolicy{Prefix: "PORTBELL_", Lookup: func(k string) (string, bool) { return "v", true }})
	if err := r.Set(Spec{Status: 700}); err == nil {
		t.Fatal("expected invalid spec to be rejected")
	}
	if r.Get().Status != 200 {
		t.Fatal("rejected spec must not replace the current one")
	}
	err := r.Set(Spec{
		Status:  201,
		Headers: []events.Header{{Name: "X-Id", Value: "${PORTBELL_ID}"}},
		Body:    "id=${PORTBELL_ID}",
		Delay:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	out := r.Render()
	if out.Status != 201 || out.Headers[0].Value != "v" || string(out.Body) != "id=v" || out.Delay != time.Second {
		t.Fatalf("unexpected render: %+v", out)
	}
}

func TestRenderSanitizesCRLFInHeaders(t *testing.T) {
	env := map[string]string{"PORTBELL_INJECT": "a\r\nb\tc\n\x00d"}
	p := EnvPolicy{Prefix: "PORTBELL_", Lookup: func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	}}
	r := New(Spec{
		Status: 200,
		Headers: []events.Header{
			{Name: "X-Inject", Value: "${PORTBELL_INJECT}"},
		},
	}, p)
	got := r.Render()
	if len(got.Headers) != 1 || got.Headers[0].Value != "ab\tcd" {
		t.Errorf("unexpected header value: %q", got.Headers[0].Value)
	}
}

func TestParseHTTP(t *testing.T) {
	in := "HTTP/1.1 201 Created\r\nContent-Type: application/json\r\nX-A:  b \r\n\r\n{\"ok\": true}\nline2"
	spec, err := ParseHTTP(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if spec.Status != 201 || len(spec.Headers) != 2 || spec.Headers[1].Value != "b" {
		t.Fatalf("unexpected spec: %+v", spec)
	}
	if spec.Body != "{\"ok\": true}\nline2" {
		t.Fatalf("body = %q", spec.Body)
	}
}

func TestParseHTTPShortForms(t *testing.T) {
	for _, in := range []string{"204", "204 No Content", "\n\nHTTP/1.0 204"} {
		spec, err := ParseHTTP(strings.NewReader(in))
		if err != nil || spec.Status != 204 {
			t.Fatalf("ParseHTTP(%q) = %+v, %v", in, spec, err)
		}
	}
}

func TestParseHTTPErrors(t *testing.T) {
	for name, in := range map[string]string{
		"empty":      "",
		"not number": "OK 200",
		"bad header": "200\nnot a header\n\nbody",
		"bad status": "999\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseHTTP(strings.NewReader(in)); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	orig := Spec{
		Status:  418,
		Headers: []events.Header{{Name: "Content-Type", Value: "text/plain"}},
		Body:    "short and stout\nsecond line",
	}
	back, err := ParseHTTP(strings.NewReader(FormatHTTP(orig)))
	if err != nil {
		t.Fatal(err)
	}
	if back.Status != orig.Status || back.Body != orig.Body || len(back.Headers) != 1 || back.Headers[0] != orig.Headers[0] {
		t.Fatalf("round trip mismatch: %+v", back)
	}
}
