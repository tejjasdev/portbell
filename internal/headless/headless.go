// Package headless prints each finished request as one JSON line, for scripts
// and CI. It reads the same event stream as the screen.
//
// The format is versioned by the "schema" field. New fields may be added within
// a schema number; existing fields are never renamed or removed.
package headless

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"
	"unicode/utf8"

	"github.com/tejjasdev/portbell/internal/events"
)

// Schema is the JSON-lines format version.
const Schema = 1

type header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type body struct {
	Text      string `json:"text,omitempty"`
	Base64    string `json:"base64,omitempty"`
	Size      int    `json:"size"`
	Truncated bool   `json:"truncated,omitempty"`
}

type requestJSON struct {
	Method  string   `json:"method"`
	Target  string   `json:"target"`
	Proto   string   `json:"proto"`
	Host    string   `json:"host"`
	Remote  string   `json:"remote"`
	Headers []header `json:"headers"`
	Body    body     `json:"body"`
}

type responseJSON struct {
	Status  int      `json:"status"`
	Headers []header `json:"headers"`
	Body    body     `json:"body"`
}

// Line is one output record.
type Line struct {
	Schema     int           `json:"schema"`
	ID         int           `json:"id"`
	Time       time.Time     `json:"time"`
	DurationMS int64         `json:"duration_ms"`
	Aborted    bool          `json:"aborted,omitempty"`
	Request    requestJSON   `json:"request"`
	Response   *responseJSON `json:"response,omitempty"`
}

// Run writes one JSON line per finished event until ctx ends or the channel
// closes. It returns the first write error.
func Run(ctx context.Context, in <-chan events.Event, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-in:
			if !ok {
				return nil
			}
			if !ev.Done {
				continue
			}
			if err := enc.Encode(ToLine(ev)); err != nil {
				return err
			}
		}
	}
}

// ToLine converts an event to its JSON record.
func ToLine(ev events.Event) Line {
	l := Line{
		Schema:     Schema,
		ID:         ev.ID,
		Time:       ev.Request.Time.UTC(),
		DurationMS: ev.Duration.Milliseconds(),
		Aborted:    ev.Aborted,
		Request: requestJSON{
			Method:  ev.Request.Method,
			Target:  ev.Request.Target,
			Proto:   ev.Request.Proto,
			Host:    ev.Request.Host,
			Remote:  ev.Request.Remote,
			Headers: headers(ev.Request.Headers),
			Body:    encodeBody(ev.Request.Body, ev.Request.BodyTruncated),
		},
	}
	if ev.Response != nil {
		l.Response = &responseJSON{
			Status:  ev.Response.Status,
			Headers: headers(ev.Response.Headers),
			Body:    encodeBody(ev.Response.Body, false),
		}
	}
	return l
}

func headers(hs []events.Header) []header {
	out := make([]header, len(hs))
	for i, h := range hs {
		out[i] = header{Name: h.Name, Value: h.Value}
	}
	return out
}

func encodeBody(b []byte, truncated bool) body {
	out := body{Size: len(b), Truncated: truncated}
	switch {
	case len(b) == 0:
	case utf8.Valid(b):
		out.Text = string(b)
	default:
		out.Base64 = base64.StdEncoding.EncodeToString(b)
	}
	return out
}
