package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Beads is the tracker's table: how a ticket's branch is named.
type Beads struct {
	BranchPattern Pattern `toml:"branch"`
}

// Branch names the branch a ticket's worktree checks out. slug is the title
// slugged, empty for a title that slugs to nothing.
func (b Beads) Branch(id, slug string) string {
	return b.pattern().tmpl.render(ticketValues(id, slug))
}

// Owns reports whether a branch is the one a ticket with this id checks out,
// whatever that ticket was titled when its worktree was made.
func (b Beads) Owns(id, branch string) bool {
	return b.pattern().owns(id, branch)
}

// An unset pattern is the compiled-in one, so a Config that never reached Load
// still names a branch.
func (b Beads) pattern() Pattern { return b.BranchPattern.or(defaultBeads.BranchPattern) }

// Pattern is a [text/template] naming a branch. Rendered with a ticket's id and
// slug it names that ticket's branch; matched against a branch with the id
// filled in and the rest wildcarded, it says whether that branch is that
// ticket's.
type Pattern struct {
	tmpl tmpl

	// matcher is the rendered pattern, every literal quoted, the id a mark, one
	// alternative per state of the slug.
	matcher string
}

// ticketValues is the data one render is given: the id, which the pattern has to
// place, and the slug, which a ticket may not have.
func ticketValues(id, slug string) map[string]any { return map[string]any{"ID": id, "Slug": slug} }

// The marks stand for a value inside a rendered arm: the id, and anything else.
// Branch names carry neither, and [regexp.QuoteMeta] leaves both alone, so what
// surrounds a mark is literal.
const (
	idMark  = "\x00"
	anyMark = "\x01"
)

// UnmarshalText reads one pattern out of a settings file. compileMatcher judges
// the values later.
func (p *Pattern) UnmarshalText(text []byte) error {
	q, err := parsePattern(string(text))
	*p = q
	return err
}

func parsePattern(text string) (Pattern, error) {
	if strings.ContainsAny(text, idMark+anyMark) {
		return Pattern{}, errors.New("carries a control byte")
	}
	// No filters: the pattern also renders into the matcher that finds a worktree again.
	t, err := parseTmpl("branch", text, nil)
	if err != nil {
		return Pattern{}, err
	}
	return Pattern{tmpl: t}, nil
}

// compileMatcher settles what the pattern matches as, or reports why it cannot
// name a branch.
func (p *Pattern) compileMatcher() error {
	alts, err := p.arms()
	if err != nil {
		return err
	}
	p.matcher = `\A(?:` + strings.Join(alts, "|") + `)\z`
	return nil
}

// arms is one quoted arm per state of the slug, or why the pattern names no
// branch at all.
func (p Pattern) arms() ([]string, error) {
	var alts []string
	// A ticket without a slug renders a branch of its own.
	for _, slug := range []string{anyMark, ""} {
		arm, err := p.tmpl.execute(ticketValues(idMark, slug))
		if err != nil {
			return nil, fmt.Errorf("%w; the values here are {{.ID}} and {{.Slug}}", err)
		}
		// A branch is found again by matching it against the pattern with the id
		// filled in, so an arm without one would stand for every ticket.
		if !strings.Contains(arm, idMark) {
			return nil, errors.New("places no {{.ID}}, so no worktree could be found by it")
		}
		if strings.HasPrefix(arm, "-") {
			return nil, errors.New("names a branch opening with a dash, which git reads as a flag")
		}
		// A pattern placing no slug renders the same arm either way.
		if alt := strings.ReplaceAll(regexp.QuoteMeta(arm), anyMark, ".+"); !slices.Contains(alts, alt) {
			alts = append(alts, alt)
		}
	}
	return alts, nil
}

// or is the pattern itself, or def where no file named one.
func (p Pattern) or(def Pattern) Pattern {
	if p.tmpl.t == nil {
		return def
	}
	return p
}

func (p Pattern) owns(id, branch string) bool {
	// Every branch the pattern names for id spells it out, so most of them are
	// ruled out without a regexp at all.
	if !strings.Contains(branch, id) {
		return false
	}
	re, err := regexp.Compile(strings.ReplaceAll(p.matcher, idMark, regexp.QuoteMeta(id)))
	return err == nil && re.MatchString(branch)
}
