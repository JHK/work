package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Branch is how a target's branch is named, one pattern per kind of target.
type Branch struct {
	TicketPattern      Pattern `toml:"ticket"`
	PullRequestPattern Pattern `toml:"pull-request"`
}

// Ticket names the branch a ticket's worktree checks out. slug is the title
// slugged, empty for a title that slugs to nothing.
func (b Branch) Ticket(id, slug string) string {
	return b.ticket().render(ticketValues.with(id, slug))
}

// PullRequest names the branch a pull request's worktree checks out, which is
// also the name the pull request is retyped as.
func (b Branch) PullRequest(number string) string {
	return b.pullRequest().render(pullRequestValues.with(number, ""))
}

// Owns reports whether a branch is the one a ticket with this id checks out,
// whatever that ticket was titled when its worktree was made.
func (b Branch) Owns(id, branch string) bool {
	return b.ticket().owns(id, branch)
}

// NumberIn reads the pull request a branch is named for. The digits are what the
// branch spells; canonicalising them is the caller's.
func (b Branch) NumberIn(branch string) (string, bool) {
	return b.pullRequest().capture(branch)
}

// An unset pattern is the compiled-in one, so a Config that never reached Load
// still names a branch.
func (b Branch) ticket() Pattern      { return b.TicketPattern.or(defaults.TicketPattern) }
func (b Branch) pullRequest() Pattern { return b.PullRequestPattern.or(defaults.PullRequestPattern) }

// Pattern is a [text/template] naming a branch. Rendered with a target's values
// it names that target's branch; matched against a branch with the identifier
// filled in and the rest wildcarded, it says whether that branch is that
// target's.
type Pattern struct {
	tmpl tmpl

	// matcher is the rendered pattern, every literal quoted, the identifier a mark,
	// one alternative per state of the optional value; numberIn compiles it for digits.
	matcher  string
	numberIn *regexp.Regexp
}

// values are what one key's pattern renders with: the value naming the target,
// which the pattern has to place, and the one a target may not have.
type values struct {
	idName       string
	optionalName string // empty where the key has none
	numbered     bool   // the identifier is digits, and reads back out of a branch
	list         string // how the values read in a refusal
}

var (
	ticketValues      = values{idName: "ID", optionalName: "Slug", list: "{{.ID}} and {{.Slug}}"}
	pullRequestValues = values{idName: "Number", numbered: true, list: "{{.Number}}"}
)

// with is the data one render is given.
func (v values) with(id, optional string) map[string]any {
	data := map[string]any{v.idName: id}
	if v.optionalName != "" {
		data[v.optionalName] = optional
	}
	return data
}

// The marks stand for a value inside a rendered arm: the identifier, and
// anything else. Branch names carry neither, and [regexp.QuoteMeta] leaves both
// alone, so what surrounds a mark is literal.
const (
	idMark  = "\x00"
	anyMark = "\x01"
)

// UnmarshalText reads one pattern out of a settings file. bind judges the values
// later.
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

// bind ties the pattern to the values its key renders with and settles what it
// matches as, or reports why it cannot name a branch.
func (p *Pattern) bind(v values) error {
	alts, err := p.alternatives(v)
	if err != nil {
		return err
	}
	p.matcher = `\A(?:` + strings.Join(alts, "|") + `)\z`

	// Nothing but the identifier varies from one match to the next, so a matcher
	// compiling here compiles at every one of them.
	numbers, err := regexp.Compile(strings.ReplaceAll(p.matcher, idMark, `([0-9]+)`))
	if err != nil {
		return err
	}
	if v.numbered {
		p.numberIn = numbers
	}
	return nil
}

// alternatives is one arm per state of the optional value, every literal quoted
// and each value left as its mark, or why the pattern names no branch at all.
func (p Pattern) alternatives(v values) ([]string, error) {
	// A target without the optional value renders a branch of its own.
	states := []string{""}
	if v.optionalName != "" {
		states = []string{anyMark, ""}
	}

	var alts []string
	for _, optional := range states {
		arm, err := p.tmpl.execute(v.with(idMark, optional))
		if err != nil {
			return nil, fmt.Errorf("%w; the values here are %s", err, v.list)
		}
		// A branch is found again by matching it against the pattern with the
		// identifier filled in, so an arm without one would stand for every target.
		if !strings.Contains(arm, idMark) {
			return nil, fmt.Errorf("places no {{.%s}}, so no worktree could be found by it", v.idName)
		}
		if strings.HasPrefix(arm, "-") {
			return nil, errors.New("names a branch opening with a dash, which git reads as a flag")
		}
		// A pattern placing no optional value renders the same arm either way.
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

func (p Pattern) render(data map[string]any) string {
	branch, err := p.tmpl.execute(data)
	if err != nil {
		return ""
	}
	return branch
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

func (p Pattern) capture(branch string) (string, bool) {
	if p.numberIn == nil {
		return "", false
	}
	m := p.numberIn.FindStringSubmatch(branch)
	if m == nil {
		return "", false
	}
	return m[1], true
}
