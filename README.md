# lazyclaude

A lazygit-style terminal UI for monitoring your local [Claude Code](https://claude.com/claude-code) sessions — see at a glance which sessions are busy, waiting for your input, or idle, along with their token usage. Watch any session's terminal live, send it instructions, and create or kill sessions, all without leaving the dashboard. Built around tmux.

```
 lazyclaude                    9 sessions · 2 busy · 1 waiting · 3 unread   14:32
╭─ Sessions · 3 unread ─────────────╮╭─ Sample refactoring session ─────────────╮
│ ❯ ! ● Sample refactoring session  ││  ● busy · 2m ago                         │
│     ◐ Fix flaky integration test  ││                                          │
│     ○ Add dark mode support       ││  model    dummy-1 · v2.x.x               │
│     ○ Write API documentation     ││  pane     work:3.1 · pid 12345           │
│                                   ││  dir      ~/projects/sample              │
│                                   ││ ── Tokens ─────────────────────────────  │
│                                   ││  context  240k                           │
│                                   ││  in / out 21k / 277k  (108 msgs)         │
│                                   ││  total    42.6M                          │
│                                   ││ ── Terminal · work:3.1 · live ─────────  │
│                                   ││  (live color preview of the session)     │
╰───────────────────────────────────╯╰──────────────────────────────────────────╯
 j/k 移動 · enter 開く · n 新規 · x 終了 · r 更新 · q quit
```

## Features

- **Session dashboard**: status (`busy` / `waiting` / `idle`), model, age,
  and token usage (current context size, cumulative in/out/cache, total)
- **Live terminal preview**: the detail pane shows the selected session's
  actual terminal, in color, refreshed twice a second
- **Fullscreen viewer** (`enter`): watch a session up close, **send it
  instructions** (`i`), or **jump to its tmux pane** (`a`)
- **Unread tracking** (`!` badge): sessions that produced output since you
  last opened them; persisted across restarts
- **Session lifecycle**: create a new Claude Code session in any directory
  (`n`) or kill one (`x`) without leaving the dashboard

## Install

```bash
go install github.com/4U-eye/lazyclaude@latest
```

Or grab a binary from [Releases](https://github.com/4U-eye/lazyclaude/releases).

## Usage

```bash
lazyclaude          # TUI
lazyclaude --list   # plain-text session listing (for scripts)
```

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | move selection |
| `enter` | open fullscreen viewer for the selected session |
| `i` (viewer) | type an instruction and send it to the session |
| `a` (viewer) | jump to the session's tmux pane |
| `n` | create a new Claude Code session (prompts for directory) |
| `x` | kill the selected session (with confirmation) |
| `g` / `G` | first / last |
| `r` | refresh now |
| `q` / `esc` | back / quit |

### Configuration

Environment variables:

| Variable | Default | Meaning |
|----------|---------|---------|
| `LAZYCLAUDE_CLAUDE_SESSION` | `claude` | tmux session that hosts new sessions created with `n` |
| `LAZYCLAUDE_CLAUDE_COMMAND` | `claude` | command typed into the new pane (runs via your shell, so aliases apply) |
| `LAZYCLAUDE_NEW_DIR` | `~` | default working directory for `n` |

## How it works

Claude Code keeps a live session registry and per-session transcripts on disk:

| Path | Contents |
|------|----------|
| `~/.claude/sessions/{pid}.json` | live registry: pid, status, cwd, updatedAt |
| `~/.claude/projects/{escaped-cwd}/{sessionId}.jsonl` | transcript: messages, token usage, session title |

lazyclaude reads these (read-only), filters out dead processes, and aggregates
token usage. Sessions are matched to tmux panes by tty, which powers the live
preview (`capture-pane -e`), instruction sending (`send-keys`), and jumping.
Read/unread state lives in `~/.local/state/lazyclaude/seen.json`.

> Note: the registry and transcript formats are undocumented internals of
> Claude Code and may change with future releases. Parsing is intentionally
> defensive.

## Requirements

- macOS / Linux
- Claude Code
- tmux (sessions running outside tmux are listed, but live preview /
  send / jump need tmux)

## Development

```bash
go test ./...   # includes tmux integration tests (skipped when tmux is absent)
go build
```

Releases are cut by pushing a `v*` tag (GoReleaser via GitHub Actions).

## License

MIT
