package headless

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/tejjasdev/portbell/internal/events"
)

func TestRunWritesOneLinePerFinishedEvent(t *testing.T) {
	store := events.NewStore(10)
	ch, cancel := store.Subscribe()

	id := store.Add(events.Request{
		Time: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC), Method: "POST", Target: "/a<b>",
		Headers: []events.Header{{Name: "X", Value: "1"}}, Body: []byte(`{"k":"<v>"}`),
	})
	store.Complete(id, events.Response{Status: 200, Body: []byte("ok")}, 12*time.Millisecond)
	id2 := store.Add(events.Request{Method: "PUT", Body: []byte{0xff, 0xfe}})
	store.Abort(id2, time.Millisecond)
	cancel() // closes the channel so Run returns after draining

	var out bytes.Buffer
	if err := Run(context.Background(), ch, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines (unfinished events are skipped), got %d:\n%s", len(lines), out.String())
	}

	var first Line
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first.Schema != 1 || first.ID != 1 || first.DurationMS != 12 || first.Request.Method != "POST" ||
		first.Request.Body.Text != `{"k":"<v>"}` || first.Response == nil || first.Response.Status != 200 {
		t.Fatalf("unexpected first line: %+v", first)
	}
	if !strings.Contains(lines[0], `"/a<b>"`) {
		t.Errorf("HTML characters should not be escaped: %s", lines[0])
	}

	var second Line
	_ = json.Unmarshal([]byte(lines[1]), &second)
	if !second.Aborted || second.Response != nil || second.Request.Body.Base64 == "" || second.Request.Body.Text != "" {
		t.Fatalf("unexpected second line: %+v", second)
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, make(chan events.Event), &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
}
