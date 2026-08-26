package main

import "testing"

func TestHasHelpFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "none", args: []string{".env"}, want: false},
		{name: "short", args: []string{HelpFlagShort}, want: true},
		{name: "long", args: []string{HelpFlagLong}, want: true},
		{name: "after a path", args: []string{".env", HelpFlagLong}, want: true},
		{name: "other flags", args: []string{"--all", "-f"}, want: false},
		{name: "no args", args: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasHelpFlag(tc.args); got != tc.want {
				t.Errorf("hasHelpFlag(%q) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
