package cli

import (
	"io"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"golang.org/x/term"
)

// Logger builds the log work says on: the level at a glance and the sentence,
// from the level given upwards, coloured where the stream is a terminal.
func Logger(w io.Writer, from slog.Leveler) *slog.Logger {
	return slog.New(tint.NewTextHandler(w, &tint.Options{
		Level:   from,
		NoColor: !terminal(w),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}))
}

// terminal reports whether w is the terminal itself rather than a file, a pipe
// or a device reading what work said.
func terminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
