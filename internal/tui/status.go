package tui

import "net/http"

func statusText(code int) string {
	if t := http.StatusText(code); t != "" {
		return t
	}
	return "Status"
}
