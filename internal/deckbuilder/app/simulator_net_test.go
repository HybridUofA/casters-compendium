package deckbuilder

import "testing"

func TestNormalizeWebSocketURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "3.86.186.199:7474", want: "ws://3.86.186.199:7474/"},
		{in: "ws://3.86.186.199:7474", want: "ws://3.86.186.199:7474/"},
		{in: "ws://3.86.186.199:7474/", want: "ws://3.86.186.199:7474/"},
		{in: "  ", want: ""},
	}
	for _, testCase := range tests {
		if got := normalizeWebSocketURL(testCase.in); got != testCase.want {
			t.Fatalf("normalizeWebSocketURL(%q) = %q; want %q", testCase.in, got, testCase.want)
		}
	}
}
