package responder

import "net/http"

// statusText returns the standard reason phrase, or "Status" for unknown codes.
func statusText(code int) string {
	if t := http.StatusText(code); t != "" {
		return t
	}
	return "Status"
}
