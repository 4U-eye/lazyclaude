# lazyclaude

A lazygit-style terminal UI for monitoring your local [Claude Code](https://claude.com/claude-code) sessions — see at a glance which sessions are busy, waiting for your input, or idle, along with their token usage. Built around tmux.

> **Status: early development.** This is a Go port of a working Python prototype.
> Phase 1 (data layer + plain listing) is done; the interactive TUI is coming next.

## What it does

```
U    PID  STATUS  MODEL      CTX     OUT    TOTAL   AGE  CWD          NAME
!  12345  busy    dummy-1   240k    277k    42.6M    2m  ~/projects   Sample refactoring session
   23456  waiting dummy-1   158k    101k     6.6M    1h  ~/projects   Fix flaky integration test
   34567  idle    dummy-2   129k     33k     4.4M    1d  ~/oss/tool   Add dark mode support
```

- **Session list** with status (`busy` / `waiting` / `idle`), unread marker (`!`),
  model, and age
- **Token usage**: current context size, cumulative output, and total consumed
  tokens per session
- **Unread tracking**: sessions that produced output since you last checked them

Planned (ported from the prototype, arriving in later phases):

- Interactive TUI (bubbletea): session list + live terminal preview of each session
- Send instructions to a session, jump to its tmux pane
- Create / kill sessions from the UI

## How it works

Claude Code keeps a live session registry and per-session transcripts on disk:

| Path | Contents |
|------|----------|
| `~/.claude/sessions/{pid}.json` | live registry: pid, status, cwd, updatedAt |
| `~/.claude/projects/{escaped-cwd}/{sessionId}.jsonl` | transcript: messages, token usage, session title |

lazyclaude reads these (read-only), filters out dead processes, and aggregates
token usage. tmux integration (pane discovery via tty matching, `capture-pane`,
`send-keys`) powers the interactive features.

> Note: these are undocumented internal formats of Claude Code and may break
> with future Claude Code releases. Parsing is intentionally defensive.

## Install

```bash
go install github.com/4U-eye/lazyclaude@latest
```

## Usage

```bash
lazyclaude --list   # plain-text session listing
lazyclaude          # TUI (not ported yet — falls back to --list output)
```

## Requirements

- macOS / Linux
- Claude Code
- tmux (for the interactive features; listing works without it)

## License

MIT
