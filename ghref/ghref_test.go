package ghref_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kitagry/htask/ghref"
)

func TestFind(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []ghref.Ref
	}{
		{
			name: "owner/repo#N",
			src:  "see kitagry/htask#12 now",
			want: []ghref.Ref{{Line: 1, Col: 5, Text: "kitagry/htask#12", Kind: ghref.OwnerRepo, Owner: "kitagry", Repo: "htask", Number: 12}},
		},
		{
			name: "repo#N",
			src:  "gokart#715 の CI",
			want: []ghref.Ref{{Line: 1, Col: 1, Text: "gokart#715", Kind: ghref.Repo, Repo: "gokart", Number: 715}},
		},
		{
			name: "PR #N",
			src:  "merge PR #42 please",
			want: []ghref.Ref{{Line: 1, Col: 7, Text: "PR #42", Kind: ghref.Prefixed, Number: 42}},
		},
		{
			name: "PRs #N and the following bare refs",
			src:  "PRs #1, #2",
			want: []ghref.Ref{
				{Line: 1, Col: 1, Text: "PRs #1", Kind: ghref.Prefixed, Number: 1},
				{Line: 1, Col: 9, Text: "#2", Kind: ghref.Bare, Number: 2},
			},
		},
		{
			name: "issue #N",
			src:  "issue #726 作成",
			want: []ghref.Ref{{Line: 1, Col: 1, Text: "issue #726", Kind: ghref.Prefixed, Number: 726}},
		},
		{
			name: "bare #N next to Japanese text",
			src:  "対応は#498で",
			want: []ghref.Ref{{Line: 1, Col: 10, Text: "#498", Kind: ghref.Bare, Number: 498}},
		},
		{
			name: "bare #N right after a link",
			src:  "[bqls#1](https://github.com/kitagry/bqls/pull/1) / #2",
			want: []ghref.Ref{{Line: 1, Col: 52, Text: "#2", Kind: ghref.Bare, Number: 2}},
		},
		{
			name: "line numbers across lines",
			src:  "first\n  - #3",
			want: []ghref.Ref{{Line: 2, Col: 5, Text: "#3", Kind: ghref.Bare, Number: 3}},
		},
		{
			name: "bare #N inside a rejected prefixed match",
			src:  "CI/PR #3",
			want: []ghref.Ref{{Line: 1, Col: 7, Text: "#3", Kind: ghref.Bare, Number: 3}},
		},
		{
			name: "bare refs separated by a slash",
			src:  "PRs #1/#2",
			want: []ghref.Ref{
				{Line: 1, Col: 1, Text: "PRs #1", Kind: ghref.Prefixed, Number: 1},
				{Line: 1, Col: 8, Text: "#2", Kind: ghref.Bare, Number: 2},
			},
		},
		{
			name: "bare refs as a range",
			src:  "#1-#3",
			want: []ghref.Ref{
				{Line: 1, Col: 1, Text: "#1", Kind: ghref.Bare, Number: 1},
				{Line: 1, Col: 4, Text: "#3", Kind: ghref.Bare, Number: 3},
			},
		},
		{
			name: "word ending in ACTIONS",
			src:  "MYACTIONS #12",
			want: []ghref.Ref{{Line: 1, Col: 11, Text: "#12", Kind: ghref.Bare, Number: 12}},
		},
		{
			name: "after a multi-line HTML comment",
			src:  "<!--\nPR #3\n--> PR #4",
			want: []ghref.Ref{{Line: 3, Col: 5, Text: "PR #4", Kind: ghref.Prefixed, Number: 4}},
		},
		{
			name: "fence inside an HTML comment does not open a code block",
			src:  "<!--\n```\n-->\n#3",
			want: []ghref.Ref{{Line: 4, Col: 1, Text: "#3", Kind: ghref.Bare, Number: 3}},
		},
		{
			name: "backtick inside an HTML comment does not hide its end",
			src:  "<!-- a ` b --> PR #3 ` c\nPR #4",
			want: []ghref.Ref{
				{Line: 1, Col: 16, Text: "PR #3", Kind: ghref.Prefixed, Number: 3},
				{Line: 2, Col: 1, Text: "PR #4", Kind: ghref.Prefixed, Number: 4},
			},
		},
		{
			name: "escaped backticks do not open a code span",
			src:  "\\`PR #3\\`",
			want: []ghref.Ref{{Line: 1, Col: 3, Text: "PR #3", Kind: ghref.Prefixed, Number: 3}},
		},
		{
			name: "after a code block in a list item",
			src:  "- ```sh\n  git log #3\n  ```\n- PR #4",
			want: []ghref.Ref{{Line: 4, Col: 3, Text: "PR #4", Kind: ghref.Prefixed, Number: 4}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, ghref.Find(tt.src)); diff != "" {
				t.Errorf("Find(%q) mismatch (-want +got):\n%s", tt.src, diff)
			}
		})
	}
}

func TestFindIgnores(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "item number", src: "- **🟠 3 A・B・C のどれ?**"},
		{name: "item number without mark", src: "- **3 これでよい?**"},
		{name: "aggregated item number", src: "- **🔴 herdr-task-go#3 merge してよい?**"},
		{name: "item ID with letters", src: "- **🟠 M1 merge してよい?**"},
		{name: "item ID with suffix", src: "- **✅ 16a 二分探索**"},
		{name: "item ID with dash", src: "- **🟢 CI-9 直す?**"},
		{name: "aggregated item ID", src: "- **🔴 herdr-task-go#16a merge してよい?**"},
		{name: "item ID list", src: "- **⏳ 90・70・91 は別タスクで決める**"},
		{name: "aggregated item ID list", src: "- **⏳ gokart-improvements#90・91 は別タスクで決める**"},
		{name: "aggregated item ID range", src: "- **✅ t#1〜t#9 済み**"},
		{name: "ACTIONS N", src: "前回の ACTIONS 12 の続き"},
		{name: "ACTIONS #N", src: "ACTIONS #12 を参照"},
		{name: "markdown link", src: "[repo#3](https://github.com/o/repo/pull/3)"},
		{name: "link text with PR", src: "[PR #3](https://github.com/o/repo/pull/3)"},
		{name: "link text with brackets", src: "[PR #3 [draft]](https://github.com/o/repo/pull/3)"},
		{name: "reference link", src: "[repo#3][1]\n\n[1]: https://github.com/o/repo/pull/3"},
		{name: "collapsed reference link", src: "[repo#3][]\n\n[repo#3]: https://github.com/o/repo/pull/3"},
		{name: "code block in a blockquote", src: "> ```\n> #3\n> ```"},
		{name: "bare URL fragment", src: "https://github.com/m3dev/gokart/issues/524#issuecomment-6888278764"},
		{name: "URL fragment with digits", src: "https://example.com/doc#12"},
		{name: "autolink", src: "<https://example.com/a#1>"},
		{name: "code span", src: "run `git log #3` here"},
		{name: "double backtick code span", src: "``a ` #3``"},
		{name: "multi-line HTML comment", src: "<!-- PR #3\n#4\n-->"},
		{name: "path ending in a bare number", src: "x/y/z#3"},
		{name: "fenced code block", src: "```\n#3 PR #4\n```"},
		{name: "tilde code block", src: "~~~sh\nrepo#5\n~~~"},
		{name: "word before hash", src: "abc#x"},
		{name: "heading", src: "## 2. Plan"},
		{name: "hash without number", src: "#tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ghref.Find(tt.src); len(got) != 0 {
				t.Errorf("Find(%q) = %+v, want none", tt.src, got)
			}
		})
	}
}

func TestFindHashAtItemHead(t *testing.T) {
	// "#688" is not an item ID, so it is a reference even at the item head.
	src := "- **✅ #688 を閉じた**"
	want := []ghref.Ref{{Line: 1, Col: 9, Text: "#688", Kind: ghref.Bare, Number: 688}}
	if diff := cmp.Diff(want, ghref.Find(src)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindAfterInlineTripleBackticks(t *testing.T) {
	// A backtick fence's info string cannot contain backticks, so this line is
	// a code span, not the start of a code block.
	src := "```#1``` then\nPR #2"
	want := []ghref.Ref{{Line: 2, Col: 1, Text: "PR #2", Kind: ghref.Prefixed, Number: 2}}
	if diff := cmp.Diff(want, ghref.Find(src)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestFindAfterCodeBlock(t *testing.T) {
	src := "```\n#1\n```\nPR #2"
	want := []ghref.Ref{{Line: 4, Col: 1, Text: "PR #2", Kind: ghref.Prefixed, Number: 2}}
	if diff := cmp.Diff(want, ghref.Find(src)); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
