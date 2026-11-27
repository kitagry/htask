# htask

CLI behind the [herdr-task](https://github.com/kitagry/dotfiles/tree/master/.claude/skills/herdr-task) skill: task workspaces, `ACTIONS.md` and `TASK.md` tooling for herdr.

The skill's shell scripts are being moved into this binary step by step. The scripts stay as the entry points and call `htask`.

## Install

```sh
mise use -g github:kitagry/htask
```

or

```sh
go install github.com/kitagry/htask/cmd/htask@latest
```

## Commands

Commands take the form `htask <noun> <verb>`.

| Command | Description |
|---|---|
| `htask version` | Print the htask version |
| `htask actions lint [--json] <ACTIONS.md>...` | Report PR / issue references that are not links. Exits 0 when clean, 1 on violations, 2 on usage errors |

### `htask actions lint`

PRs and issues in `ACTIONS.md` must be written as `[repo#N](https://github.com/<owner>/<repo>/pull/N)` (`/issues/N` for issues). The lint prints one line per reference that is not a link:

```
ACTIONS.md:3: PR #42 is not a link
```

Flagged: `owner/repo#N`, `repo#N`, `PR #N` / `PRs #N` / `issue #N`, and bare `#N`.
Not flagged: the item ID at the head of an item, in the form `([A-Z]+-?)?[0-9]+[a-z]?` (`- **🟠 3 ...`, `M1`, `16a`, `CI-9`), a list of them (`10・11`, `1〜9`), or `<task>#<ID>` after a status mark in aggregated files; `ACTIONS 12` / `ACTIONS #12`; anything inside Markdown links, URLs, code spans, code blocks and HTML comments. Anything else at the item head, such as `- **✅ #688 ...`, is still flagged.

The detection lives in the [`ghref`](https://pkg.go.dev/github.com/kitagry/htask/ghref) package so other tools can share it.

## License

MIT
