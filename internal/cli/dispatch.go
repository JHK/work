package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// dispatch puts the bare form behind the verb it is a shortcut for: a first
// word naming no verb, and no word at all, are [goCommand]'s. Cobra reads its
// own completion request off that same position, so it is left where it is;
// every tab press is one.
func dispatch(root *cobra.Command, args []string) []string {
	if rootsOwn(root, args) {
		return args
	}
	return append([]string{"go"}, args...)
}

// rootsOwn reports whether the first word is the root's to answer: a flag it
// declares, the completion request, or a command it carries.
func rootsOwn(root *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return false
	}
	word := args[0]
	if strings.HasPrefix(word, "-") {
		return rootFlag(root, word)
	}
	if word == cobra.ShellCompRequestCmd || word == cobra.ShellCompNoDescRequestCmd {
		return true
	}
	// The one word, never the rest: Find reads a flag's value off the words after it.
	found, _, _ := root.Find(args[:1])
	return found != root
}

// rootFlag reports whether a flag in the bare position is the root's rather than
// one of switch's, a shorthand and a spelled-out value counting the same.
func rootFlag(root *cobra.Command, word string) bool {
	name, _, _ := strings.Cut(strings.TrimLeft(word, "-"), "=")
	if strings.HasPrefix(word, "--") {
		return root.Flags().Lookup(name) != nil
	}
	return len(name) == 1 && root.Flags().ShorthandLookup(name) != nil
}
