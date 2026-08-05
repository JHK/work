package cli

import (
	"io"
	"testing"
)

func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		want   options
		target string
	}{
		{"no identifier", []string{"--shell"}, options{shell: true}, ""},
		{"flags after the identifier", []string{"bd-1", "--start"}, options{start: true}, "bd-1"},
		{"a flag value is not an identifier", []string{"bd-1", "--start", "--model", "opus"}, options{start: true, model: "opus"}, "bd-1"},
		{"joined value", []string{"--start", "--effort=high", "bd-1"}, options{start: true, effort: "high"}, "bd-1"},
		{"flags before the identifier", []string{"--shell", "bd-1"}, options{shell: true}, "bd-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got options
			var target string
			err := run(tt.args, func(o options, id string) error {
				got, target = o, id
				return nil
			})
			if err != nil {
				t.Fatalf("Execute(%q): %v", tt.args, err)
			}
			if got != tt.want || target != tt.target {
				t.Errorf("Execute(%q) = %+v, %q; want %+v, %q", tt.args, got, target, tt.want, tt.target)
			}
		})
	}
}

func TestCommandRejects(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"--start and --shell", []string{"bd-1", "--start", "--shell"}},
		{"--model without --start", []string{"bd-1", "--model", "opus"}},
		{"--effort without --start", []string{"bd-1", "--effort", "high"}},
		{"two identifiers", []string{"bd-1", "bd-2"}},
		{"unknown flag", []string{"bd-1", "--turbo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args, func(options, string) error {
				t.Error("ran despite invalid flags")
				return nil
			})
			if err == nil {
				t.Errorf("Execute(%q): want an error", tt.args)
			}
		})
	}
}

func run(args []string, f func(options, string) error) error {
	cmd := command(f)
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd.Execute()
}
