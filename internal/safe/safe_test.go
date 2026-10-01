package safe

import "testing"

func TestText(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain", "hello\nworld\t!", "hello\nworld\t!"},
		{"ansi escape", "a\x1b[31mred\x1b[0m", `a\x1b[31mred\x1b[0m`},
		{"carriage return", "x\ry", `x\x0dy`},
		{"nul", "a\x00b", `a\x00b`},
		{"invalid utf8", "a\xffb", `a\xffb`},
		{"c1 control", "a\u009bb", `a\x9bb`},
		{"bidi override", "a\u202eb", `a\u202eb`},
		{"unicode ok", "नमस्ते 你好", "नमस्ते 你好"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Text(tc.in); got != tc.want {
				t.Fatalf("Text(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestLine(t *testing.T) {
	if got := Line("a\nb\tc"); got != `a\nb\tc` {
		t.Fatalf("got %q", got)
	}
}
