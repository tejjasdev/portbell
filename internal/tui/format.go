package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/safe"
)

const (
	maxBodyShown = 64 << 10 // bytes of a body rendered on screen
	maxPretty    = 256 << 10
	hexBytes     = 128
)

// fit cuts s to width columns and pads it with spaces to exactly width.
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	if w := lipgloss.Width(s); w < width {
		s += strings.Repeat(" ", width-w)
	}
	return s
}

// wrapLine breaks s into pieces of at most width runes. It is plain-text only;
// callers pass text that has already been made safe.
func wrapLine(s string, width int) []string {
	s = strings.ReplaceAll(s, "\t", "    ")
	if width < 1 {
		return []string{s}
	}
	r := []rune(s)
	if len(r) <= width {
		return []string{s}
	}
	var out []string
	for len(r) > width {
		out = append(out, string(r[:width]))
		r = r[width:]
	}
	return append(out, string(r))
}

func wrapAll(lines []string, width int) []string {
	var out []string
	for _, l := range lines {
		out = append(out, wrapLine(l, width)...)
	}
	return out
}

// headerValue returns the first value of a header, case-insensitively.
func headerValue(hs []events.Header, name string) string {
	for _, h := range hs {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// bodyLines renders a body as readable text lines. Every line is already safe
// to print in a terminal.
func bodyLines(body []byte, contentType string, truncated bool) []string {
	if len(body) == 0 {
		return []string{"(empty)"}
	}
	var lines []string
	shown := body
	cut := 0
	if len(shown) > maxBodyShown {
		cut = len(shown) - maxBodyShown
		shown = shown[:maxBodyShown]
	}

	switch {
	case !utf8.Valid(shown):
		lines = hexDump(shown)
	default:
		lines = textLines(shown, contentType, len(body) <= maxPretty)
	}
	if cut > 0 {
		lines = append(lines, fmt.Sprintf("… %d more bytes not shown", cut))
	}
	if truncated {
		lines = append(lines, "… body was cut at the size limit")
	}
	return lines
}

func textLines(b []byte, contentType string, canPretty bool) []string {
	ct := strings.ToLower(contentType)
	trimmed := bytes.TrimSpace(b)

	if canPretty && (strings.Contains(ct, "json") || (len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '['))) && json.Valid(trimmed) {
		var out bytes.Buffer
		if err := json.Indent(&out, trimmed, "", "  "); err == nil {
			return strings.Split(safe.Text(out.String()), "\n")
		}
	}
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		if vals, err := url.ParseQuery(string(b)); err == nil && len(vals) > 0 {
			keys := make([]string, 0, len(vals))
			for k := range vals {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var lines []string
			for _, k := range keys {
				for _, v := range vals[k] {
					lines = append(lines, safe.Line(k)+" = "+safe.Line(v))
				}
			}
			return lines
		}
	}
	return strings.Split(safe.Text(strings.ReplaceAll(string(b), "\r\n", "\n")), "\n")
}

func hexDump(b []byte) []string {
	lines := []string{fmt.Sprintf("binary data, %d bytes", len(b))}
	n := len(b)
	if n > hexBytes {
		n = hexBytes
	}
	for i := 0; i < n; i += 16 {
		end := i + 16
		if end > n {
			end = n
		}
		var hex, txt strings.Builder
		for _, c := range b[i:end] {
			fmt.Fprintf(&hex, "%02x ", c)
			if c >= 0x20 && c < 0x7f {
				txt.WriteByte(c)
			} else {
				txt.WriteByte('.')
			}
		}
		lines = append(lines, fmt.Sprintf("%04x  %-48s %s", i, hex.String(), txt.String()))
	}
	if len(b) > n {
		lines = append(lines, fmt.Sprintf("… %d more bytes not shown", len(b)-n))
	}
	return lines
}

// detailLines builds the text of the detail pane for one event.
func detailLines(ev events.Event) []string {
	r := ev.Request
	var l []string
	l = append(l, fmt.Sprintf("%s %s %s", safe.Line(r.Method), safe.Line(r.Target), safe.Line(r.Proto)))
	meta := fmt.Sprintf("from %s at %s · body %s", safe.Line(r.Remote), r.Time.Format("15:04:05"), size(len(r.Body)))
	if ev.Done {
		meta += " · took " + took(ev.Duration)
	}
	l = append(l, meta, "", "Headers")
	for _, h := range r.Headers {
		l = append(l, "  "+safe.Line(h.Name)+": "+safe.Line(h.Value))
	}
	l = append(l, "", "Body")
	for _, b := range bodyLines(r.Body, headerValue(r.Headers, "Content-Type"), r.BodyTruncated) {
		l = append(l, "  "+b)
	}

	l = append(l, "")
	switch {
	case ev.Aborted:
		l = append(l, "Reply: none, the client disconnected first")
	case ev.Response == nil:
		l = append(l, "Reply: waiting…")
	default:
		resp := ev.Response
		l = append(l, fmt.Sprintf("Reply: %d %s", resp.Status, statusText(resp.Status)))
		for _, h := range resp.Headers {
			l = append(l, "  "+safe.Line(h.Name)+": "+safe.Line(h.Value))
		}
		l = append(l, "")
		for _, b := range bodyLines(resp.Body, headerValue(resp.Headers, "Content-Type"), false) {
			l = append(l, "  "+b)
		}
	}
	return l
}

// took formats how long a request took, without showing a misleading "0s".
func took(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return d.Round(10 * time.Millisecond).String()
}

func size(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
}
