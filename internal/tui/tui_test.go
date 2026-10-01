package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/responder"
)

func newTestModel(t *testing.T) (Model, *responder.Responder) {
	t.Helper()
	store := events.NewStore(50)
	resp := responder.New(responder.Default(), responder.EnvPolicy{})
	m := New(Options{Store: store, Resp: resp, URL: "http://127.0.0.1:8080", Redact: true, History: 50})
	return update(m, tea.WindowSizeMsg{Width: 110, Height: 30}), resp
}

func update(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "ctrl+s":
		return tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func request(id int, method, target string) events.Event {
	return events.Event{
		ID: id,
		Request: events.Request{
			Time: time.Date(2026, 10, 1, 12, 0, id, 0, time.UTC), Method: method, Target: target,
			Proto: "HTTP/1.1", Remote: "127.0.0.1:5000",
			Headers: []events.Header{{Name: "Content-Type", Value: "application/json"}},
			Body:    []byte(`{"event":"paid"}`),
		},
	}
}

func TestEmptyStateExplainsWhatToDo(t *testing.T) {
	m, _ := newTestModel(t)
	out := m.render()
	for _, want := range []string{"Waiting for requests on http://127.0.0.1:8080", "curl http://127.0.0.1:8080/hello", "? for help"} {
		if !strings.Contains(out, want) {
			t.Errorf("empty screen is missing %q", want)
		}
	}
}

func TestRequestsAppearAndSelectionFollowsNewest(t *testing.T) {
	m, _ := newTestModel(t)
	m = update(m, eventMsg(request(1, "POST", "/hook")))
	m = update(m, eventMsg(request(2, "GET", "/ping")))
	if m.sel != 1 {
		t.Fatalf("selection should follow the newest request, got %d", m.sel)
	}
	out := m.render()
	for _, want := range []string{"POST", "/hook", "GET", "/ping", "Requests (2)", `"event": "paid"`} {
		if !strings.Contains(out, want) {
			t.Errorf("screen is missing %q", want)
		}
	}

	m = update(m, key("k")) // move up: stops following
	if m.sel != 0 || m.follow {
		t.Fatalf("after k: sel=%d follow=%v", m.sel, m.follow)
	}
	m = update(m, eventMsg(request(3, "PUT", "/x")))
	if m.sel != 0 {
		t.Fatalf("selection must stay put when the user moved away, got %d", m.sel)
	}
	m = update(m, key("G"))
	if m.sel != 2 || !m.follow {
		t.Fatalf("G should jump to the newest and follow: sel=%d follow=%v", m.sel, m.follow)
	}
}

func TestCompletedEventUpdatesInPlace(t *testing.T) {
	m, _ := newTestModel(t)
	ev := request(1, "GET", "/a")
	m = update(m, eventMsg(ev))
	ev.Done = true
	ev.Response = &events.Response{Status: 404, Body: []byte("nope")}
	m = update(m, eventMsg(ev))
	if len(m.evs) != 1 {
		t.Fatalf("update must replace the event, have %d", len(m.evs))
	}
	if out := m.render(); !strings.Contains(out, "404") || !strings.Contains(out, "Not Found") {
		t.Errorf("reply status missing:\n%s", out)
	}
}

func TestUntrustedTextIsEscaped(t *testing.T) {
	m, _ := newTestModel(t)
	ev := request(1, "GET", "/\x1b[2Jevil")
	ev.Request.Headers = append(ev.Request.Headers, events.Header{Name: "X-Evil", Value: "a\x1b]0;pwned\x07b"})
	ev.Request.Body = []byte("line\x1b[31mred")
	ev.Request.Headers[0].Value = "text/plain"
	m = update(m, eventMsg(ev))
	out := m.render()
	if strings.Contains(out, "\x1b[2J") || strings.Contains(out, "\x1b]0;pwned") || strings.Contains(out, "\x1b[31mred") {
		t.Fatal("raw escape sequence from a request reached the screen")
	}
	if !strings.Contains(out, `\x1b[2Jevil`) {
		t.Errorf("escaped form should be visible:\n%s", out)
	}
}

func TestSmallWindowShowsMessage(t *testing.T) {
	m, _ := newTestModel(t)
	m = update(m, tea.WindowSizeMsg{Width: 30, Height: 8})
	if out := m.render(); !strings.Contains(out, "Window too small") {
		t.Fatalf("expected a size message, got:\n%s", out)
	}
}

func TestEveryRowFitsTheWindow(t *testing.T) {
	m, _ := newTestModel(t)
	long := request(1, "POST", "/"+strings.Repeat("very-long-path/", 20))
	long.Request.Body = []byte(strings.Repeat("0123456789", 100))
	m = update(m, eventMsg(long))
	for _, size := range [][2]int{{110, 30}, {60, 14}, {200, 50}} {
		m = update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		lines := strings.Split(m.render(), "\n")
		if len(lines) > size[1] {
			t.Errorf("%dx%d: %d lines, more than the window height", size[0], size[1], len(lines))
		}
	}
}

func TestEditReplyApplies(t *testing.T) {
	m, resp := newTestModel(t)
	m = update(m, key("e"))
	if m.mode != modeEdit {
		t.Fatal("e should open the editor")
	}
	m.editor.SetValue("503 Service Unavailable\nRetry-After: 5\n\nbusy")
	m.delay.SetValue("250ms")
	m = update(m, key("ctrl+s"))
	if m.mode != modeNormal || m.editErr != "" {
		t.Fatalf("apply failed: mode=%v err=%q", m.mode, m.editErr)
	}
	got := resp.Get()
	if got.Status != 503 || got.Delay != 250*time.Millisecond || got.Body != "busy" || got.Headers[0].Name != "Retry-After" {
		t.Fatalf("reply not updated: %+v", got)
	}
}

func TestEditRejectsBadInputAndKeepsOldReply(t *testing.T) {
	for name, tc := range map[string]struct{ text, delay string }{
		"bad status": {"abc", "0s"},
		"bad delay":  {"200\n\nok", "soon"},
		"bad range":  {"700\n\nok", "0s"},
	} {
		t.Run(name, func(t *testing.T) {
			m, resp := newTestModel(t)
			m = update(m, key("e"))
			m.editor.SetValue(tc.text)
			m.delay.SetValue(tc.delay)
			m = update(m, key("ctrl+s"))
			if m.mode != modeEdit || m.editErr == "" {
				t.Fatalf("should stay in the editor with an error: mode=%v err=%q", m.mode, m.editErr)
			}
			if resp.Get().Status != 200 {
				t.Fatal("a rejected edit must not change the reply")
			}
			m = update(m, key("esc"))
			if m.mode != modeNormal {
				t.Fatal("esc should leave the editor")
			}
		})
	}
}

func TestHelpOpensAndAnyKeyCloses(t *testing.T) {
	m, _ := newTestModel(t)
	m = update(m, key("?"))
	if m.mode != modeHelp || !strings.Contains(m.render(), "In the reply editor") {
		t.Fatal("help should open")
	}
	m = update(m, key("x"))
	if m.mode != modeNormal {
		t.Fatal("any key should close help")
	}
}

func TestQuit(t *testing.T) {
	m, _ := newTestModel(t)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("q should return a quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q should quit")
	}
}

func TestPublicBindWarningIsShown(t *testing.T) {
	store := events.NewStore(5)
	m := New(Options{Store: store, Resp: responder.New(responder.Default(), responder.EnvPolicy{}), URL: "http://x", Warning: "bound to 0.0.0.0: anyone can connect"})
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if !strings.Contains(m.render(), "anyone can connect") {
		t.Fatal("warning should be visible")
	}
}

func TestBodyFormatting(t *testing.T) {
	json := strings.Join(bodyLines([]byte(`{"a":[1,2]}`), "application/json", false), "\n")
	if !strings.Contains(json, "  \"a\": [") {
		t.Errorf("json should be indented:\n%s", json)
	}
	form := strings.Join(bodyLines([]byte("b=2&a=1"), "application/x-www-form-urlencoded", false), "\n")
	if form != "a = 1\nb = 2" {
		t.Errorf("form body: %q", form)
	}
	bin := strings.Join(bodyLines([]byte{0xff, 0x00, 0x41}, "", false), "\n")
	if !strings.Contains(bin, "binary data, 3 bytes") || !strings.Contains(bin, "ff 00 41") {
		t.Errorf("binary body: %q", bin)
	}
	if got := bodyLines(nil, "", false); got[0] != "(empty)" {
		t.Errorf("empty body: %v", got)
	}
	cut := bodyLines([]byte("abc"), "text/plain", true)
	if cut[len(cut)-1] != "… body was cut at the size limit" {
		t.Errorf("truncation note missing: %v", cut)
	}
}

func TestTookFormatting(t *testing.T) {
	for in, want := range map[time.Duration]string{
		0:                       "<1ms",
		400 * time.Microsecond:  "<1ms",
		12 * time.Millisecond:   "12ms",
		1534 * time.Millisecond: "1.53s",
	} {
		if got := took(in); got != want {
			t.Errorf("took(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestCtrlCQuitsInEveryMode(t *testing.T) {
	ctrlC := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	for name, setup := range map[string]func(Model) Model{
		"normal": func(m Model) Model { return m },
		"edit":   func(m Model) Model { return update(m, key("e")) },
		"help":   func(m Model) Model { return update(m, key("?")) },
	} {
		t.Run(name, func(t *testing.T) {
			m, _ := newTestModel(t)
			m = setup(m)
			_, cmd := m.Update(ctrlC)
			if cmd == nil {
				t.Fatal("Ctrl+C should quit")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("Ctrl+C should return a quit message")
			}
		})
	}
}

func TestEditAndHelpScreensFitTheWindow(t *testing.T) {
	m, _ := newTestModel(t)
	m = update(m, eventMsg(request(1, "GET", "/a")))
	for _, size := range [][2]int{{60, 14}, {80, 24}, {110, 30}, {220, 60}} {
		for name, open := range map[string]string{"edit": "e", "help": "?"} {
			mm := update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			mm = update(mm, key(open))
			lines := strings.Split(mm.render(), "\n")
			if len(lines) > size[1] {
				t.Errorf("%s at %dx%d: %d lines exceed the window height", name, size[0], size[1], len(lines))
			}
		}
	}
}

func TestMouseIsNotCaptured(t *testing.T) {
	m, _ := newTestModel(t)
	if got := m.View().MouseMode; got != tea.MouseModeNone {
		t.Fatalf("mouse mode = %v; capturing the mouse breaks text selection", got)
	}
}
