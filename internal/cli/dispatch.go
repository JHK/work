package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// dispatch puts the bare form behind go: a first word naming no verb, and no
// word at all. Cobra reads its completion request off that position, so it is
// left where it is.
func dispatch(root *cobra.Command, args []string) []string {
	if rootOwns(root, args) {
		return args
	}
	return append([]string{"go"}, args...)
}

// rootOwns reports whether the first word is the root's to answer: a flag of its
// own, the completion request, or a command it carries.
func rootOwns(root *cobra.Command, args []string) bool {
	rest := pastRootFlags(root, args)
	if len(rest) == 0 {
		return false
	}
	word := rest[0]
	if strings.HasPrefix(word, "-") {
		return declared(root.LocalNonPersistentFlags(), word) != nil
	}
	if word == cobra.ShellCompRequestCmd || word == cobra.ShellCompNoDescRequestCmd {
		return true
	}
	// The one word, never the rest: Find reads a flag's value off the words after it.
	found, _, _ := root.Find(rest[:1])
	return found != root
}

// pastRootFlags is what is left of the words once the flags the root hands down,
// and the words they take their values from, are stepped over. A flag the root
// hands down holds no position, and neither does the word behind it.
func pastRootFlags(root *cobra.Command, args []string) []string {
	// R2 leaves the handed-down flag spelled out, so no shorthand reaches here.
	for len(args) > 0 {
		handed := declared(root.PersistentFlags(), args[0])
		if handed == nil {
			break
		}
		takes := !strings.Contains(args[0], "=") && handed.NoOptDefVal == ""
		args = args[1:]
		if takes && len(args) > 0 {
			args = args[1:]
		}
	}
	return args
}

// declared is the flag a set spells that word, a shorthand and a spelled-out
// value counting the same, and nil where the set spells none.
func declared(set *pflag.FlagSet, word string) *pflag.Flag {
	name, _, _ := strings.Cut(strings.TrimLeft(word, "-"), "=")
	if strings.HasPrefix(word, "--") {
		return set.Lookup(name)
	}
	if len(name) != 1 {
		return nil
	}
	return set.ShorthandLookup(name)
}
