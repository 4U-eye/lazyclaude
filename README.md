# lazyclaude

A lazygit-style terminal UI for monitoring your local [Claude Code](https://claude.com/claude-code) sessions — see at a glance which sessions are busy, waiting for your input, or idle, along with their token usage. Watch any session's terminal live, send it instructions, and create or kill sessions, all without leaving the dashboard. Built around tmux.

```
 lazyclaude                    9 sessions · 2 busy · 1 waiting · 3 unread   14:32
╭─ Sessions · M3 E1 · 3 unread ─────╮╭─ Sample refactoring session ─────────────╮
│ ❯ ! ● M-01 Sample refactoring ses ││  ● busy · 2m ago · [ MAX ]               │
│     ◐ E-01 Fix flaky integration  ││                                          │
│     ○ M-02 Add dark mode support  ││  model    dummy-1 · v2.x.x               │
│     ○ M-03 Write API documentation││  pane     work:3.1 · pid 12345           │
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
- **Multiple installations**: Claude Code stores its state under
  `CLAUDE_CONFIG_DIR`, so a personal plan and an enterprise one can live side
  by side. lazyclaude scans every configured installation and colours each
  one distinctly — unit codes (`M-01` / `E-01`), a plan pill in the detail
  pane, and a per-plan count in the panel subtitle
- **Live terminal preview**: the detail pane shows the selected session's
  actual terminal, in color, refreshed twice a second
- **Fullscreen viewer** (`enter`): watch a session up close, **send it
  instructions** (`i`), or **jump to its tmux pane** (`a`)
- **Unread tracking** (`!` badge): sessions that produced output since you
  last opened them; persisted across restarts
- **Session lifecycle**: create a new Claude Code session in any directory
  (`n`, picking which plan to launch it in) or kill one (`x`) without leaving
  the dashboard

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
| `n` | create a new Claude Code session (picks the plan when more than one installation is configured, then prompts for directory) |
| `A` | jump to the assistant session, launching it first when it is not running (needs `[assistant]` in the config) |
| `x` | kill the selected session (with confirmation) |
| `g` / `G` | first / last |
| `r` | refresh now |
| `q` / `esc` | back / quit |

### Configuration

Settings load from `~/.config/lazyclaude/config.toml` (honors `XDG_CONFIG_HOME`);
environment variables override the file.

```toml
# default working directory when creating a session with `n`
new_dir = "~/projects"

# tmux session that hosts new sessions (default: claude)
claude_session = "claude"

# launch command for the first profile when it doesn't set one of its own
claude_command = "claude"
```

| Key | Env override | Default |
|-----|--------------|---------|
| `new_dir` | `LAZYCLAUDE_NEW_DIR` | `~` |
| `claude_session` | `LAZYCLAUDE_CLAUDE_SESSION` | `claude` |
| `claude_command` | `LAZYCLAUDE_CLAUDE_COMMAND` | `claude` |

#### Profiles (installations / plans)

By default lazyclaude watches two installations: `~/.claude` (labelled `MAX`,
cyan) and `~/.claude-enterprise` (labelled `ENT`, orange). A profile whose
directory doesn't exist is simply skipped, so the defaults are safe even with
only one installation.

Declaring any `[[profiles]]` replaces both defaults:

```toml
[[profiles]]
key = "max"              # stable id; defaults to the lowercased label
label = "MAX"            # shown in the UI; its first letter becomes the unit code prefix
config_dir = "~/.claude" # the installation's CLAUDE_CONFIG_DIR
command = "claude"       # typed into the new pane via your shell, so aliases apply
tmux_session = "claude"  # defaults to claude_session
color = "45"             # 256-color index or hex; drives the code and pill colour

[[profiles]]
key = "ent"
label = "ENT"
config_dir = "~/.claude-enterprise"
command = "claude-e"
color = "208"
```

Order matters: it is the display order, and the number keys in the `n` picker.

#### Assistant session

A resident session you can reach with one key. `A` looks for a live session
whose name (set by `--name`) matches `name` and jumps to it; when there is
none, it launches `{profile command} {args} --name {name}` in `dir` and jumps
there. The key is disabled until `name` is set.

```toml
[assistant]
name = "assistant"           # session name to look for / launch with
dir = "~/work"               # default: new_dir
profile = "max"              # profile key to launch in; default: the first profile
args = "--agent assistant"   # e.g. run a custom agent as the main thread
```

## How it works

Claude Code keeps a live session registry and per-session transcripts on disk,
under `CLAUDE_CONFIG_DIR` (`~/.claude` unless overridden):

| Path | Contents |
|------|----------|
| `{config_dir}/sessions/{pid}.json` | live registry: pid, status, cwd, updatedAt |
| `{config_dir}/projects/{escaped-cwd}/{sessionId}.jsonl` | transcript: messages, token usage, session title |

lazyclaude reads these (read-only) for every configured profile, filters out
dead processes, and aggregates token usage into one list. Sessions are matched
to tmux panes by tty — independent of which installation they came from — which
powers the live preview (`capture-pane -e`), instruction sending (`send-keys`),
and jumping. Read/unread state lives in `~/.local/state/lazyclaude/seen.json`,
keyed by session id, so it is shared across installations.

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
