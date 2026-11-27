// Package ghref finds GitHub pull request and issue references in Markdown
// that are written as plain text instead of links.
//
// The herdr-task skill requires PRs and issues to be written as
// [repo#N](https://github.com/<owner>/<repo>/pull/N). Find reports the
// references that break that rule, so they can be flagged (htask actions lint)
// or turned into links by callers that know which repository they belong to.
package ghref

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// Kind tells how a reference names its repository.
type Kind int

const (
	// Bare is "#N" with nothing in front.
	Bare Kind = iota
	// Prefixed is "PR #N", "PRs #N" or "issue #N"; the repository is implied.
	Prefixed
	// Repo is "repo#N".
	Repo
	// OwnerRepo is "owner/repo#N".
	OwnerRepo
)

func (k Kind) String() string {
	switch k {
	case Bare:
		return "bare"
	case Prefixed:
		return "prefixed"
	case Repo:
		return "repo"
	case OwnerRepo:
		return "owner/repo"
	}
	return "Kind(" + strconv.Itoa(int(k)) + ")"
}

// Ref is a reference written as plain text.
type Ref struct {
	// Line and Col are 1-based; Col counts bytes.
	Line, Col int
	// Text is the reference as written, e.g. "PR #42".
	Text string
	Kind Kind
	// Owner and Repo are empty when the text does not name them.
	Owner, Repo string
	Number      int
}

var (
	refPattern = regexp.MustCompile(
		`(?i:\b(PRs?|issues?) ?#([0-9]+))` +
			`|([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+)#([0-9]+)` +
			`|([A-Za-z0-9_.-]+)#([0-9]+)` +
			`|#([0-9]+)`)

	// The ID at the head of an ACTIONS.md item ("- **🟠 3 ...", "M1", "16a",
	// "CI-9", or a list such as "10・11") is the item's own ID, not a
	// reference. Aggregated files write it as "<task>#3", which is only told
	// apart from "repo#3" by the status mark in front of it. Anything else
	// there, such as "#688", is still a reference.
	itemNumberPattern = regexp.MustCompile(`^\s*[-*+]\s+\*\*\s*(?:[^\s\w*]+\s+(` + itemList(`(?:[a-z0-9][a-z0-9._-]*#)?`+itemID) + `)|(` + itemList(itemID) + `))(?:\s|\*|$)`)

	// Inline links and full or collapsed reference links. The link text may
	// hold one level of brackets, as in "[PR #3 [draft]](...)".
	linkPattern = regexp.MustCompile(`!?\[(?:[^\[\]]|\[[^\[\]]*\])*\](?:\([^)]*\)|\[[^\]]*\])`)
	// The label of a link reference definition, "[repo#3]: https://...".
	linkDefPattern  = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:`)
	autolinkPattern = regexp.MustCompile(`<[A-Za-z][A-Za-z0-9+.-]*:[^\s<>]*>`)
	urlPattern      = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[!-~]+`)

	// containerPattern matches the blockquote and list markers in front of a
	// line, so that "- ```sh" and "> ```" still open a code block.
	containerPattern = regexp.MustCompile(`^(?:\s*(?:>|(?:[-*+]|[0-9]{1,9}[.)])(?:\s|$)))*\s*`)
)

const itemID = `(?:[A-Z]+-?)?[0-9]+[a-z]?`

// itemList matches one item ID or several joined by "・", "〜", "~", "," or "/".
func itemList(id string) string {
	return id + `(?:\s*[・〜~,/]\s*` + id + `)*`
}

// Find returns the references in a Markdown document that are not links.
// Text inside links, URLs, code spans, fenced code blocks and HTML comments is
// skipped.
func Find(src string) []Ref {
	var refs []Ref
	var fence string
	inComment := false
	for i, line := range strings.Split(src, "\n") {
		if fence != "" {
			if isFenceClose(line, fence) {
				fence = ""
			}
			continue
		}
		// A fence written inside an HTML comment does not open a code block.
		if !inComment {
			if f := fenceOpen(line); f != "" {
				fence = f
				continue
			}
		}
		var lineRefs []Ref
		lineRefs, inComment = findInLine(line, i+1, inComment)
		refs = append(refs, lineRefs...)
	}
	return refs
}

func findInLine(line string, lineNo int, inComment bool) ([]Ref, bool) {
	masked, inComment := mask(line, inComment)
	var refs []Ref
	// Matches are searched one at a time instead of with FindAll, so that a
	// rejected match ("CI/PR #3") does not swallow a reference inside it.
	for pos := 0; pos < len(masked); {
		m := refPattern.FindStringSubmatchIndex(masked[pos:])
		if m == nil {
			break
		}
		for k := range m {
			if m[k] >= 0 {
				m[k] += pos
			}
		}
		ref, ok := toRef(masked, m, lineNo)
		if !ok {
			pos = m[0] + 1
			continue
		}
		refs = append(refs, ref)
		pos = m[1]
	}
	return refs, inComment
}

// toRef builds the Ref for a refPattern match, or reports false when the
// match is not a reference.
func toRef(masked string, m []int, lineNo int) (Ref, bool) {
	start, end := m[0], m[1]
	if end < len(masked) && isWordByte(masked[end]) {
		return Ref{}, false
	}
	group := func(n int) string {
		if m[2*n] < 0 {
			return ""
		}
		return masked[m[2*n]:m[2*n+1]]
	}
	ref := Ref{Line: lineNo, Col: start + 1, Text: masked[start:end]}
	switch {
	case m[2] >= 0:
		ref.Kind = Prefixed
		ref.Number = atoi(group(2))
	case m[6] >= 0:
		ref.Kind = OwnerRepo
		ref.Owner, ref.Repo, ref.Number = group(3), group(4), atoi(group(5))
	case m[12] >= 0:
		ref.Kind = Repo
		ref.Repo, ref.Number = group(6), atoi(group(7))
		if ref.Repo == "ACTIONS" {
			return Ref{}, false
		}
	default:
		ref.Kind = Bare
		ref.Number = atoi(group(8))
		if start > 0 && !isBareLeftBoundary(masked[start-1]) {
			return Ref{}, false
		}
		before, ok := strings.CutSuffix(strings.TrimRight(masked[:start], " "), "ACTIONS")
		if ok && (before == "" || !isWordByte(before[len(before)-1])) {
			return Ref{}, false
		}
		return ref, true
	}
	if start > 0 && !isLeftBoundary(masked[start-1]) {
		return Ref{}, false
	}
	return ref, true
}

var maskPatterns = []*regexp.Regexp{linkPattern, linkDefPattern, autolinkPattern, urlPattern}

// mask blanks out the parts of a line that must not be searched, keeping
// byte offsets unchanged so columns still point into the original line.
// inComment tells whether the line starts inside an HTML comment; the second
// result tells whether the next line does.
func mask(line string, inComment bool) (string, bool) {
	b := []byte(line)
	inComment = maskCodeAndComments(b, inComment)
	for _, p := range maskPatterns {
		for _, loc := range p.FindAllIndex(b, -1) {
			blank(b, loc[0], loc[1])
		}
	}
	if m := itemNumberPattern.FindSubmatchIndex(b); m != nil {
		for _, g := range []int{2, 4} {
			if m[g] >= 0 {
				blank(b, m[g], m[g+1])
			}
		}
	}
	return string(b), inComment
}

func blank(b []byte, start, end int) {
	for i := start; i < end; i++ {
		b[i] = ' '
	}
}

// maskCodeAndComments blanks code spans and HTML comments in b and reports
// whether the next line starts inside a comment. Whichever of the two opens
// first wins, as in CommonMark, so a backtick inside a comment does not hide
// the comment's end. A span opened by a run of N backticks is closed only by
// another run of exactly N backticks.
func maskCodeAndComments(b []byte, inComment bool) bool {
	i := 0
	if inComment {
		end := bytes.Index(b, []byte("-->"))
		if end < 0 {
			blank(b, 0, len(b))
			return true
		}
		i = end + len("-->")
		blank(b, 0, i)
	}
	for i < len(b) {
		switch {
		case b[i] == '\\':
			i += 2
		case bytes.HasPrefix(b[i:], []byte("<!--")):
			end := bytes.Index(b[i+len("<!--"):], []byte("-->"))
			if end < 0 {
				blank(b, i, len(b))
				return true
			}
			e := i + len("<!--") + end + len("-->")
			blank(b, i, e)
			i = e
		case b[i] == '`':
			n := backtickRun(b, i)
			if e := codeSpanEnd(b, i+n, n); e >= 0 {
				blank(b, i, e)
				i = e
			} else {
				i += n
			}
		default:
			i++
		}
	}
	return false
}

// codeSpanEnd returns the end of the run of exactly n backticks at or after
// from, or -1 when there is none.
func codeSpanEnd(b []byte, from, n int) int {
	for j := from; j < len(b); {
		if b[j] != '`' {
			j++
			continue
		}
		m := backtickRun(b, j)
		if m == n {
			return j + m
		}
		j += m
	}
	return -1
}

func backtickRun(s []byte, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}

// stripContainers drops the blockquote and list markers in front of line.
func stripContainers(line string) string {
	return line[len(containerPattern.FindString(line)):]
}

func fenceOpen(line string) string {
	t := stripContainers(line)
	for _, c := range []byte{'`', '~'} {
		n := 0
		for n < len(t) && t[n] == c {
			n++
		}
		// A backtick fence's info string cannot contain backticks.
		if n >= 3 && (c == '~' || !strings.Contains(t[n:], "`")) {
			return t[:n]
		}
	}
	return ""
}

func isFenceClose(line, fence string) bool {
	t := strings.TrimSpace(stripContainers(line))
	return strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == ""
}

func isWordByte(c byte) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// isLeftBoundary rejects matches that continue a path, a version, an HTML
// entity ("&#123;") or another reference.
func isLeftBoundary(c byte) bool {
	return !isWordByte(c) && !strings.ContainsRune("./-#&", rune(c))
}

// isBareLeftBoundary is looser than isLeftBoundary: "#1/#2" and "#1-#3" list
// references rather than continue a path or a version.
func isBareLeftBoundary(c byte) bool {
	return !isWordByte(c) && !strings.ContainsRune(".#&", rune(c))
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
