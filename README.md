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

## License

MIT
