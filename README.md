# Vibeship

Vibeship is a TUI (terminal user interface) cockpit for Claude Code sessions. It provides real-time visual feedback on your Claude Code usage — showing token rates, cost, rate-limit consumption, active skills and agents, todos, and a transcript of recent events — all in a compact dashboard that sits beside your terminal.

## Install

```bash
go install github.com/francis/vibeship@latest
```

This requires Go 1.22 or later. The binary will be placed at `$GOPATH/bin/vibeship` (or `$HOME/go/bin/vibeship` if `GOPATH` is not set).

## Setup

Vibeship ingests data from the Claude Code statusline. Configure Claude Code to pipe its statusline output through `vibeship collect` by adding the following to your Claude Code configuration (`~/.claude/settings.json` or project-level `.claude/settings.json`):

```json
{
  "statusLine": {
    "command": "vibeship collect"
  }
}
```

The `vibeship collect` command reads statusline JSON lines from stdin, stores usage snapshots in a local SQLite database (`~/.vibeship/data.db`), and returns a forwarded statusline string so your prompt remains unaffected.

## Usage

Start the dashboard:

```bash
vibeship
```

This launches the Bubble Tea TUI. The left panel shows a real-time animation (spaceship or DJ theme) visualizing token activity. The right panel displays metrics, recent activity, active agents, and todos.

Use the overlay panels to inspect skills/plugins (`s`) or check your project scope (`d`).

### Collect mode (piped from Claude Code)

The `collect` subcommand is not meant to be run directly. It is invoked automatically by Claude Code when configured as the statusline command.

```bash
vibeship collect
```

## Keybindings

| Key | Action |
|-----|--------|
| `t`   | Toggle theme (Spaceship / DJ) |
| `s`   | Toggle skills & plugins sidebar |
| `d`   | Toggle think panel (scope check / brainstorming) |
| `r`   | Force refresh data |
| `?`   | Show help |
| `q`   | Quit |
| `esc` | Close overlay |

## Project Scope

Optionally create a `SCOPE.md` file in your project root. Vibeship reads it and checks whether your current Claude Code session is staying on track.
