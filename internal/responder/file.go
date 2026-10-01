package responder

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tejjasdev/portbell/internal/events"
)

// MaxFileBytes is the largest reply file ParseHTTP accepts.
const MaxFileBytes = 1 << 20

// ParseHTTP reads a reply written like a plain HTTP response:
//
//	HTTP/1.1 201 Created        (or "201 Created", or just "201")
//	Content-Type: application/json
//
//	{"ok": true}
//
// The delay is not part of the file; set it with a flag, the config file or the
// screen.
func ParseHTTP(r io.Reader) (Spec, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxFileBytes+1))
	if err != nil {
		return Spec{}, err
	}
	if len(data) > MaxFileBytes {
		return Spec{}, fmt.Errorf("reply is larger than %d bytes", MaxFileBytes)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 64*1024), MaxFileBytes)

	var spec Spec
	lineNo := 0
	haveStatus := false
	inBody := false
	var body []string
	for sc.Scan() {
		line := sc.Text()
		lineNo++
		if inBody {
			body = append(body, line)
			continue
		}
		if !haveStatus {
			if strings.TrimSpace(line) == "" {
				continue // allow blank lines before the status line
			}
			status, err := parseStatusLine(line)
			if err != nil {
				return Spec{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			spec.Status = status
			haveStatus = true
			continue
		}
		if line == "" {
			inBody = true
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return Spec{}, fmt.Errorf("line %d: expected \"Name: value\", got %q", lineNo, line)
		}
		spec.Headers = append(spec.Headers, events.Header{Name: strings.TrimSpace(name), Value: strings.TrimSpace(value)})
	}
	if err := sc.Err(); err != nil {
		return Spec{}, err
	}
	if !haveStatus {
		return Spec{}, fmt.Errorf("empty reply: the first line must be a status such as \"200 OK\"")
	}
	spec.Body = strings.Join(body, "\n")
	if err := spec.Validate(); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func parseStatusLine(line string) (int, error) {
	fields := strings.Fields(line)
	if len(fields) > 0 && strings.HasPrefix(strings.ToUpper(fields[0]), "HTTP/") {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return 0, fmt.Errorf("missing status code in %q", line)
	}
	code, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, fmt.Errorf("status code %q is not a number", fields[0])
	}
	return code, nil
}

// FormatHTTP writes a Spec in the format ParseHTTP reads.
func FormatHTTP(s Spec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "HTTP/1.1 %d %s\n", s.Status, statusText(s.Status))
	for _, h := range s.Headers {
		fmt.Fprintf(&b, "%s: %s\n", h.Name, h.Value)
	}
	b.WriteString("\n")
	b.WriteString(s.Body)
	return b.String()
}
