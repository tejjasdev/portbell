// Package tui is the terminal screen: a request list, a detail pane, and an
// editor for the reply. It reads events and changes the reply; it never talks
// to the HTTP server directly.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tejjasdev/portbell/internal/brand"
	"github.com/tejjasdev/portbell/internal/events"
	"github.com/tejjasdev/portbell/internal/export"
	"github.com/tejjasdev/portbell/internal/responder"
	"github.com/tejjasdev/portbell/internal/safe"
)

// Minimum terminal size the layout needs.
const (
	minWidth  = 60
	minHeight = 14
)

// SaveDir is the folder where "save request" writes files.
const SaveDir = brand.SaveDir

// Options connects the screen to the rest of the program.
type Options struct {
	Store   *events.Store
	Resp    *responder.Responder
	Sub     <-chan events.Event // updates from Store.Subscribe
	URL     string              // address shown in the header
	Warning string              // shown in red, for example a public bind address
	Redact  bool                // hide secrets in copied and saved requests
	History int                 // how many events to keep on screen
}

type pane int

const (
	paneList pane = iota
	paneDetail
)

type mode int

const (
	modeNormal mode = iota
	modeEdit
	modeHelp
)

type (
	eventMsg  events.Event
	closedMsg struct{}
)

// Model is the Bubble Tea model.
type Model struct {
	opts Options

	evs     []events.Event
	sel     int
	listTop int
	follow  bool
	focus   pane
	mode    mode

	detailOff int
	w, h      int
	status    string

	editor    textarea.Model
	delay     textinput.Model
	editField int // 0 = reply text, 1 = delay
	editErr   string
}

// New returns a Model ready to run.
func New(opts Options) Model {
	if opts.History < 1 {
		opts.History = events.DefaultCapacity
	}
	ed := textarea.New()
	ed.ShowLineNumbers = false
	ed.CharLimit = 0
	ed.MaxHeight = 0
	di := textinput.New()
	di.Prompt = ""
	di.CharLimit = 16

	m := Model{opts: opts, follow: true, editor: ed, delay: di, w: 100, h: 30}
	if opts.Store != nil {
		m.evs = opts.Store.Snapshot()
		if len(m.evs) > 0 {
			m.sel = len(m.evs) - 1
		}
	}
	return m
}

// Run starts the screen and blocks until the user quits or ctx is cancelled.
func Run(ctx context.Context, opts Options) error {
	_, err := tea.NewProgram(New(opts), tea.WithContext(ctx)).Run()
	if err != nil && ctx.Err() != nil && errors.Is(err, tea.ErrProgramKilled) {
		return nil // we were asked to stop: not a failure
	}
	return err
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return waitEvent(m.opts.Sub)
}

func waitEvent(ch <-chan events.Event) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return closedMsg{}
		}
		return eventMsg(ev)
	}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.resizeEditor()
		m.keepSelectionVisible()
		return m, nil

	case eventMsg:
		m.upsert(events.Event(msg))
		return m, waitEvent(m.opts.Sub)

	case closedMsg:
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit // always works, in every mode
		}
		m.status = ""
		switch m.mode {
		case modeEdit:
			return m.updateEdit(msg)
		case modeHelp:
			m.mode = modeNormal
			return m, nil
		default:
			return m.updateNormal(msg)
		}

	}

	if m.mode == modeEdit { // paste, cursor blink and other messages go to the editor
		return m.forwardToEditor(msg)
	}
	return m, nil
}

func (m *Model) upsert(ev events.Event) {
	for i := len(m.evs) - 1; i >= 0; i-- {
		if m.evs[i].ID == ev.ID {
			m.evs[i] = ev
			return
		}
	}
	m.evs = append(m.evs, ev)
	if len(m.evs) > m.opts.History {
		m.evs = m.evs[len(m.evs)-m.opts.History:]
		if m.sel > 0 {
			m.sel--
		}
	}
	if m.follow {
		m.sel = len(m.evs) - 1
		m.detailOff = 0
	}
	m.keepSelectionVisible()
}

func (m Model) updateNormal(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.mode = modeHelp
	case "tab":
		if m.focus == paneList {
			m.focus = paneDetail
		} else {
			m.focus = paneList
		}
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "pgup":
		m.move(-m.pageSize())
	case "pgdown":
		m.move(m.pageSize())
	case "home", "g":
		m.moveTo(0)
	case "end", "G":
		m.moveTo(len(m.evs) - 1)
	case "e":
		m.startEdit()
		return m, m.editor.Focus()
	case "c":
		if ev, ok := m.selected(); ok {
			m.status = fmt.Sprintf("Copied curl command for request #%d (needs a terminal that supports clipboard access)", ev.ID)
			return m, tea.SetClipboard(export.Curl(ev.Request, "http", m.opts.Redact))
		}
	case "s":
		if ev, ok := m.selected(); ok {
			path, err := export.SaveRequest(SaveDir, ev, m.opts.Redact)
			if err != nil {
				m.status = "Could not save: " + err.Error()
			} else {
				m.status = "Saved " + path
			}
		}
	}
	return m, nil
}

func (m *Model) move(delta int) {
	if m.focus == paneList {
		m.moveSelection(delta)
	} else {
		m.scrollDetail(delta)
	}
}

func (m *Model) moveSelection(delta int) {
	m.moveTo(m.sel + delta)
}

func (m *Model) moveTo(i int) {
	if m.focus == paneDetail {
		if i <= 0 {
			m.detailOff = 0
		} else {
			m.detailOff = 1 << 30 // clamped when drawn
		}
		return
	}
	if len(m.evs) == 0 {
		return
	}
	if i < 0 {
		i = 0
	}
	if i > len(m.evs)-1 {
		i = len(m.evs) - 1
	}
	if i != m.sel {
		m.detailOff = 0
	}
	m.sel = i
	m.follow = i == len(m.evs)-1
	m.keepSelectionVisible()
}

func (m *Model) scrollDetail(delta int) {
	m.detailOff += delta
	if m.detailOff < 0 {
		m.detailOff = 0
	}
}

func (m Model) pageSize() int {
	_, _, rows := m.layout()
	if rows < 2 {
		return 1
	}
	return rows - 1
}

func (m Model) selected() (events.Event, bool) {
	if len(m.evs) == 0 || m.sel < 0 || m.sel >= len(m.evs) {
		return events.Event{}, false
	}
	return m.evs[m.sel], true
}

func (m *Model) keepSelectionVisible() {
	_, _, rows := m.layout()
	if rows < 1 {
		return
	}
	if m.sel < m.listTop {
		m.listTop = m.sel
	}
	if m.sel >= m.listTop+rows {
		m.listTop = m.sel - rows + 1
	}
	if m.listTop < 0 {
		m.listTop = 0
	}
}

// ---- reply editor ----------------------------------------------------------

func (m *Model) startEdit() {
	spec := m.opts.Resp.Get()
	m.editor.SetValue(responder.FormatHTTP(spec))
	m.delay.SetValue(spec.Delay.String())
	m.editField = 0
	m.editErr = ""
	m.mode = modeEdit
	m.resizeEditor()
}

func (m *Model) resizeEditor() {
	_, _, rows := m.layout()
	m.editor.SetWidth(max(20, m.w-4))
	m.editor.SetHeight(max(3, rows-4))
	m.delay.SetWidth(16)
}

func (m Model) updateEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.editor.Blur()
		m.delay.Blur()
		return m, nil
	case "ctrl+s":
		return m.applyEdit()
	case "tab":
		if m.editField == 0 {
			m.editField = 1
			m.editor.Blur()
			return m, m.delay.Focus()
		}
		m.editField = 0
		m.delay.Blur()
		return m, m.editor.Focus()
	case "enter":
		if m.editField == 1 {
			return m.applyEdit()
		}
	}
	return m.forwardToEditor(msg)
}

func (m Model) forwardToEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.editField == 0 {
		m.editor, cmd = m.editor.Update(msg)
	} else {
		m.delay, cmd = m.delay.Update(msg)
	}
	return m, cmd
}

func (m Model) applyEdit() (tea.Model, tea.Cmd) {
	spec, err := responder.ParseHTTP(strings.NewReader(m.editor.Value()))
	if err != nil {
		m.editErr = err.Error()
		return m, nil
	}
	if d := strings.TrimSpace(m.delay.Value()); d != "" {
		spec.Delay, err = time.ParseDuration(d)
		if err != nil {
			m.editErr = fmt.Sprintf("delay %q is not a duration like 500ms or 2s", d)
			return m, nil
		}
	}
	if err := m.opts.Resp.Set(spec); err != nil {
		m.editErr = err.Error()
		return m, nil
	}
	m.mode = modeNormal
	m.editor.Blur()
	m.delay.Blur()
	m.status = "Reply updated. It applies to the next request."
	return m, nil
}

// ---- drawing ----------------------------------------------------------------

var (
	styleTitle = lipgloss.NewStyle().Bold(true)
	styleDim   = lipgloss.NewStyle().Faint(true)
	styleSel   = lipgloss.NewStyle().Reverse(true)
	styleWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	styleOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

// View implements tea.Model.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	// The mouse is deliberately not captured: that keeps normal text selection
	// and copy working in the terminal.
	return v
}

// layout returns the list width, the detail width and the number of content
// rows in the main area (the title row is not counted).
func (m Model) layout() (listW, detailW, rows int) {
	listW = min(40, max(24, m.w/3))
	detailW = m.w - listW - 1
	top := 1
	if m.opts.Warning != "" {
		top++
	}
	rows = m.h - top - 2 /* reply bar + hint bar */ - 1 /* pane title */
	return listW, detailW, max(rows, 1)
}

func (m Model) render() string {
	if m.w < minWidth || m.h < minHeight {
		return fmt.Sprintf("Window too small: need at least %dx%d, have %dx%d.\nMake it larger, or press q to quit.",
			minWidth, minHeight, m.w, m.h)
	}
	var lines []string
	lines = append(lines, fit(styleTitle.Render(" "+brand.Name+" ")+styleOK.Render("● ")+"listening on "+m.opts.URL, m.w))
	if m.opts.Warning != "" {
		lines = append(lines, styleWarn.Render(fit(" ! "+m.opts.Warning, m.w)))
	}

	_, _, rows := m.layout()
	switch {
	case m.mode == modeHelp:
		lines = append(lines, m.helpLines(rows+1)...)
	case m.mode == modeEdit:
		lines = append(lines, m.editLines(rows+1)...)
	case len(m.evs) == 0:
		lines = append(lines, m.emptyLines(rows+1)...)
	default:
		lines = append(lines, m.mainLines(rows)...)
	}

	lines = append(lines, m.replyBar(), m.hintBar())
	return strings.Join(lines, "\n")
}

func (m Model) mainLines(rows int) []string {
	lw, dw, _ := m.layout()
	ev := m.evs[m.sel]

	listTitle := fmt.Sprintf(" Requests (%d)", len(m.evs))
	detailTitle := fmt.Sprintf(" Request #%d", ev.ID)
	if m.focus == paneList {
		listTitle = "▸" + listTitle[1:]
	} else {
		detailTitle = "▸" + detailTitle[1:]
	}
	out := []string{styleTitle.Render(fit(listTitle, lw)) + styleDim.Render("│") + styleTitle.Render(fit(detailTitle, dw))}

	detail := wrapAll(detailLines(ev), dw-2)
	maxOff := max(0, len(detail)-rows)
	off := min(m.detailOff, maxOff)

	for i := 0; i < rows; i++ {
		left := strings.Repeat(" ", lw)
		if idx := m.listTop + i; idx < len(m.evs) {
			text := fit(m.listLine(m.evs[idx], lw), lw)
			if idx == m.sel {
				text = styleSel.Render(text)
			}
			left = text
		}
		right := strings.Repeat(" ", dw)
		if off+i < len(detail) {
			right = fit(" "+detail[off+i], dw)
		}
		if i == 0 && off > 0 {
			right = fit(" ↑ more", dw)
		}
		if i == rows-1 && off < maxOff {
			right = fit(" ↓ more", dw)
		}
		out = append(out, left+styleDim.Render("│")+right)
	}
	return out
}

func (m Model) listLine(ev events.Event, width int) string {
	state := "…"
	switch {
	case ev.Aborted:
		state = "✗"
	case ev.Response != nil:
		state = fmt.Sprintf("%d", ev.Response.Status)
	}
	head := fmt.Sprintf(" %s %-6s %s", ev.Request.Time.Format("15:04:05"), safe.Line(ev.Request.Method), safe.Line(ev.Request.Target))
	room := width - len(state) - 1
	if room < 1 {
		return head
	}
	return fit(head, room) + " " + state
}

func (m Model) replyBar() string {
	s := m.opts.Resp.Get()
	text := fmt.Sprintf(" Reply: %d %s · delay %s · body %s", s.Status, statusText(s.Status), s.Delay, size(len(s.Body)))
	return styleTitle.Render(fit(text, m.w))
}

func (m Model) hintBar() string {
	if m.status != "" {
		return styleOK.Render(fit(" "+safe.Line(m.status), m.w))
	}
	var hint string
	switch m.mode {
	case modeEdit:
		hint = " Ctrl+S apply · Esc cancel · Tab switch field"
	case modeHelp:
		hint = " Press any key to close help"
	default:
		hint = " Tab pane · ↑↓ move · e edit reply · c copy curl · s save request · ? help · q quit"
	}
	return styleDim.Render(fit(hint, m.w))
}

func (m Model) emptyLines(rows int) []string {
	text := []string{
		"",
		"  Waiting for requests on " + m.opts.URL,
		"",
		"  Try it from another terminal:",
		"",
		"    curl " + m.opts.URL + "/hello",
		"    curl -X POST " + m.opts.URL + "/hook -d '{\"hello\":\"world\"}'",
		"",
		"  Every request appears here. Press e to change the reply,",
		"  ? for help, q to quit.",
	}
	return padRows(text, rows, m.w)
}

func (m Model) helpLines(rows int) []string {
	text := []string{
		"",
		"  Keys",
		"",
		"    Tab           switch between the request list and the detail pane",
		"    ↑ ↓  j k      move in the list, or scroll the detail pane",
		"    PgUp PgDn     move a page",
		"    g  G          jump to the first or last item",
		"    e             edit the reply (status, headers, body, delay)",
		"    c             copy the selected request as a curl command",
		"    s             save the selected request to ./" + SaveDir,
		"    ?             this help",
		"    q  Ctrl+C     quit",
		"",
		"  In the reply editor",
		"",
		"    Ctrl+S        apply     Esc  cancel     Tab  switch between text and delay",
		"    First line is the status, then headers, a blank line, then the body.",
		"",
		"  Secrets (Authorization, Cookie, ...) are hidden in copied and saved requests.",
		"  Control characters in requests are shown as \\x1b-style escapes.",
	}
	return padRows(text, rows, m.w)
}

func (m Model) editLines(rows int) []string {
	out := []string{
		styleTitle.Render(fit(" Edit reply", m.w)),
	}
	out = append(out, strings.Split(m.editor.View(), "\n")...)
	delayLabel := "  Delay: "
	out = append(out, delayLabel+m.delay.View()+styleDim.Render("   examples: 0s  250ms  2s"))
	if m.editErr != "" {
		out = append(out, styleWarn.Render(fit("  ! "+safe.Line(m.editErr), m.w)))
	}
	return padRows(out, rows, m.w)
}

// padRows makes sure there are exactly rows lines, each cut to width.
func padRows(lines []string, rows, width int) []string {
	out := make([]string, 0, rows)
	for i := 0; i < rows; i++ {
		if i < len(lines) {
			out = append(out, fit(lines[i], width))
		} else {
			out = append(out, strings.Repeat(" ", width))
		}
	}
	return out
}
