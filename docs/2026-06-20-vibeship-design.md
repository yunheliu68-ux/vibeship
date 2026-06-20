# Vibeship — Design Spec

> 一个终端里给 vibe coder 的实时驾驶舱——看见 token 在烧、知道 agent 在干嘛、感知自己有没有跑偏。

**Status:** Design Approved · **Date:** 2026-06-20

---

## 1. Product Overview

### 1.1 Positioning

Vibeship is a standalone TUI dashboard that pairs with Claude Code. It visualizes token consumption, active tools/skills/plugins, running agents, and task progress — all in a single glance. It is NOT a replacement for Claude Code; it is a companion cockpit that runs in a separate terminal window.

### 1.2 Core Job

> Keep a solo vibe coder from losing control — see what's burning, what's running, whether they're drifting off-scope, and talk through product decisions before writing code.

### 1.3 Two Equal Pillars

Vibeship has two co-equal functions, both in Phase 1:

1. **Runtime Cockpit** — Token animation, active tools/skills/agents, task progress. Always-on dashboard.
2. **Thinking Co-pilot** — Brainstorming, scope checks, product thinking, develop-order guardrails. On-demand via `d` key.

### 1.3 User

- **Primary:** The author (francis), using Claude Code in terminal
- **Future:** Other terminal-based vibe coders

---

## 2. Architecture

### 2.1 Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Language | Go | Single binary, fast, Bubble Tea ecosystem |
| TUI framework | Bubble Tea + Lip Gloss | Mature, component-based, terminal-native |
| Animation | Bubble Tea ticks + custom render | Frame-based animation within terminal constraints |
| Data storage | SQLite (mattn/go-sqlite3) | Lightweight, zero config, shared between collect & TUI |
| Distribution | `go install` / Homebrew / prebuilt binary | Simple, no runtime deps |

### 2.2 Data Flow

```
Claude Code
  │
  ├── statusline stdin JSON (every ~300ms)
  │     └── {model, context_window, rate_limits, cost, ...}
  │           │
  │           ▼
  │     vibeship collect (tiny sidecar, embedded in main binary)
  │           │
  │           ▼ SQLite INSERT
  │     ~/.vibeship/data.db ──────────┐
  │                                   │
  ├── ~/.claude/sessions/{id}.jsonl  │
  │     (transcript polling, every 2s)│
  │           │                       │
  │           ▼                       │
  │     goroutine: parse tools,       │
  │     skills, agents, todos         │
  │           │                       │
  │           ▼ SQLite INSERT         │
  │     ~/.vibeship/data.db ──────────┤
  │                                   │
  ▼                                   ▼
┌──────────────────────────────────────────┐
│           Vibeship TUI Process            │
│                                          │
│  ┌────────┐  ┌────────┐  ┌────────┐     │
│  │ Theme  │  │Layout  │  │ Keymap │     │
│  │ Engine │  │ Manager│  │Manager │     │
│  └────────┘  └────────┘  └────────┘     │
│                                          │
│  Bubble Tea Update → View → Render       │
└──────────────────────────────────────────┘
```

### 2.3 How It Runs Alongside Claude Code

1. User starts Claude Code in one terminal
2. User starts `vibeship` in another terminal (or same terminal, different tmux pane)
3. Claude Code pushes data via statusline pipe → `vibeship collect` stores to SQLite
4. Vibeship TUI reads SQLite and renders the dashboard
5. When Claude Code exits, Vibeship keeps the last known state until user quits

**Claude Code settings.json snippet:**

```json
{
  "statusLine": {
    "type": "command",
    "command": "vibeship collect | <existing-statusline-command>"
  }
}
```

`vibeship collect` acts as a transparent pipe — it reads stdin JSON, extracts and stores relevant fields to SQLite, then passes the original JSON unchanged to stdout so the existing statusline keeps working.

---

## 3. TUI Layout

### 3.1 Default View (No Overlays)

```
┌──────────────────────────────────────────────────────────┐
│  Vibeship · my-project git:(main*)         🚀 曲速  42%  │  top bar
├────────────────────────────────┬─────────────────────────┤
│                                │  ┌───────────────────┐  │
│                                │  │ $3.42  45% ctx  🟢│  │  metrics card
│                                │  └───────────────────┘  │
│                                │  ┌───────────────────┐  │
│        Token Animation         │  │ ◐ Write: auth.ts  │  │
│         (70% width,            │  │ 🧩 brainstorming  │  │  activity card
│          centered)              │  │ 🔌 github         │  │
│                                │  └───────────────────┘  │
│                                │  ┌───────────────────┐  │
│                                │  │ 🤖 explore [haiku]│  │
│        Theme-dependent:        │  │ ○ security queued │  │  agents card
│        Spaceship or DJ         │  └───────────────────┘  │
│                                │  ┌───────────────────┐  │
│                                │  │ 📋 40% · 2/5 done │  │  todos card
│                                │  └───────────────────┘  │
└────────────────────────────────┴─────────────────────────┘

  t=主题  s=skills  ?=帮助  q=退出
```

### 3.2 Information Architecture

**Always visible (no interaction needed):**

| Zone | Content | Data Source |
|------|---------|-------------|
| Top bar | Project name, git branch, theme indicator | transcript JSONL |
| Left 70% | Token animation (theme-dependent) | statusline stdin |
| Metrics card | Cost, context %, rate limit status | statusline stdin |
| Activity card | Active tool, skill, MCP server | transcript JSONL |
| Agents card | Running/submitted agents with duration | transcript JSONL |
| Todos card | Progress bar + top 3 pending items | transcript JSONL |

**On-demand (key-triggered overlays):**

| Key | Overlay | Content |
|-----|---------|---------|
| `s` | Skills sidebar | All available skills (✓ active / — idle), plugins, recommended skill |
| `?` | Help overlay | Keyboard shortcuts reference |
| `d` | Thinking co-pilot | Brainstorming, scope check, develop-order guard |

### 3.3 `s` — Skills & Plugins Sidebar

```
┌──────────────────────────────────────┬──────────────────┐
│                                      │ 🧩 Skills (12)   │
│                                      │                  │
│        Token Animation               │ superpowers:     │
│                                      │   brainstorming ✓│
│                                      │   writing-plans   │
│                                      │   tdd             │
│                                      │ code-review      │
│                                      │ frontend-design   │
│                                      │ ...              │
│                                      │ ──────────────── │
│                                      │ 🔌 Plugins (8)   │
│                                      │ github ✓          │
│                                      │ playwright ✓      │
│                                      │ figma             │
│                                      │ pencil            │
│                                      │ ...              │
│                                      │ ──────────────── │
│                                      │ 💡 Recommended    │
│                                      │ → code-review     │
│                                      │   (auth.ts changed│
│                                      │    in last 5 min) │
└──────────────────────────────────────┴──────────────────┘
```

Sidebar slides in from the right, pushing the right-side cards to overlay mode. Press `s` again to dismiss.

**"Recommended skill" logic (lightweight):**
- If a file was recently edited (`Write`/`Edit` detected in transcript) → suggest `code-review`
- If a new Git branch was checked out → suggest `writing-plans`
- If `Bash` errors detected → suggest `systematic-debugging`
- Otherwise → no recommendation shown

This is a simple rules engine — NOT an LLM call. Zero cost, zero latency.

### 3.4 `d` — Thinking Co-pilot Overlay (Phase 1)

Pressing `d` opens a full-height side panel on the right (overlaying the info cards). This is the "thinking co-pilot" — it helps the user pause and think before coding.

```
┌──────────────────────────────────────┬──────────────────┐
│                                      │ 💭 思路副驾       │
│                                      │                  │
│        Token Animation               │ 📋 当前 Scope    │
│                                      │ auth module      │
│                                      │ login + signup   │
│                                      │ ─────────────── │
│                                      │ 📊 最近改动      │
│                                      │ src/auth.ts  ✏️   │
│                                      │ src/db/user.go ✏️ │
│                                      │ ─────────────── │
│                                      │ ⚡ 快速检查      │
│                                      │                  │
│                                      │ 1. 当前改动在    │
│                                      │    scope 内吗？   │
│                                      │                  │
│                                      │ 2. 先做前端还是   │
│                                      │   后端？          │
│                                      │                  │
│                                      │ 3. 数据结构定义   │
│                                      │   好了吗？         │
│                                      │                  │
│                                      │ ─────────────── │
│                                      │ 💬 输入你的问题…  │
│                                      │ [Enter → 提示]   │
└──────────────────────────────────────┴──────────────────┘
```

**How it works:**

1. Vibeship reads `SCOPE.md` or `PRD.md` from the project root (user maintains this file)
2. Pressing `d` shows: current scope → recent file changes → auto-generated check questions → free-text input
3. User types a question at the bottom and presses Enter
4. Vibeship displays the question prominently — user switches to Claude Code and pastes/asks it there
5. Vibeship logs Q&A to `~/.vibeship/decisions.jsonl` for future reference (why-trail)

**Auto-generated check questions (rules engine, not LLM, zero cost):**

| Trigger | Question Generated |
|---------|-------------------|
| `Write`/`Edit` on files NOT in SCOPE.md | "⚠️ 这些文件不在 scope 里，确定要继续吗？" |
| `Write` detected but no schema/migration changed | "💡 数据结构定好了吗？建议先定义 schema 再写代码。" |
| Frontend files changed, no backend (or vice versa) | "💡 前后端接口对齐了吗？建议先确认 API contract。" |
| No TODO completed in last 10 min | "💡 是不是卡住了？要不要换个角度聊聊？" |
| New dependency added (go.mod, package.json change) | "💡 新加了依赖，确认过选型吗？有无更轻的替代？" |
| Always shown | Current SCOPE.md goals as reference |

**Scope file format (SCOPE.md or PRD.md):**

A simple markdown file the user maintains. Vibeship parses it for:
- `## Files` section → list of in-scope file patterns
- `## Goals` section → what this phase should achieve
- `## Out of Scope` section → explicitly excluded
- `## Develop Order` section → frontend→backend→database dependency order

Example:
```markdown
# SCOPE: User Authentication

## Goals
- Login with email/password
- Signup with email verification
- Session management

## Files
- src/auth/*
- src/middleware/session.ts
- db/migrations/001_users.sql

## Out of Scope
- OAuth / social login
- Password reset flow
- Role-based access

## Develop Order
1. Database schema (users table)
2. Backend auth endpoints
3. Frontend login/signup forms
```

**Cost:** Zero LLM calls. Pure rules engine + file reading.

---

## 4. Visual Themes

Two themes, switchable with `t` key. The theme affects ONLY the left 70% animation area. Right-side cards share a neutral color scheme.

### 4.1 🚀 Spaceship Cockpit (Default)

**Color palette:**
- Background: `#0a0e1a` (deep navy blue-black)
- Primary: `#00d4ff` (cyan)
- Secondary: `#ff6600` (orange, for warnings)
- Text: `#c8d6e5` (cool white)
- Dim: `#576574` (muted gray-blue)

**Animation elements:**
- Center: Speedometer gauge — dial arc, tick marks, pointer needle
- Pointer moves in real-time based on output token rate
- Background: particle field (stars) — density and speed proportional to token rate
- Bottom strip: "warp speed line" — horizontal dash stream, speed = token rate

**States:**
- 🟢 Normal pace → cyan pointer, sparse stars, calm speed line
- 🟡 Approaching budget → orange pointer, medium stars
- 🔴 Over budget / near limit → red pointer, dense fast stars, speed line pulsing

### 4.2 🎵 DJ Mixer

**Color palette:**
- Background: `#1a0a2e` (deep purple-black)
- Primary: `#00ff88` (neon green-cyan)
- Secondary: `#ff00ff` (magenta)
- Text: `#e8d5f5` (warm white)
- Dim: `#6b5b7a` (muted purple)

**Animation elements:**
- Top spectrum bar: input token visualization — columns jump with input activity
- Center: breathing halo around the token rate number
  - Breath frequency = budget consumption % (faster = closer to limit)
  - Halo color shifts from cyan → orange → red
- Bottom spectrum bar: output token visualization
- Spectrum columns: height = recent N-second token volume, color gradient cool→hot

**States:**
- 🟢 Idle/low → low columns, slow halo, cool colors
- 🟡 Active → jumping columns, moderate halo
- 🔴 Peak → full-height columns, rapid halo pulse, warm colors

### 4.3 Shared UI (Both Themes)

Right-side cards use a muted neutral palette that works with both themes:
- Card border: subtle gray
- Active indicator: green `◐` spinner
- Done: `✓` checkmark
- Labels: dim color
- Warning/alert: yellow or red accent (theme-agnostic)

---

## 5. Data Schema

### 5.1 Statusline Ingest (from `vibeship collect`)

Stored per tick in `usage_snapshots`:

```sql
CREATE TABLE usage_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp TEXT NOT NULL,          -- ISO 8601
  session_id TEXT NOT NULL,
  model_display_name TEXT,
  context_used_pct REAL,
  input_tokens INTEGER,
  output_tokens INTEGER,
  cache_create_tokens INTEGER,
  cache_read_tokens INTEGER,
  total_cost_usd REAL,
  five_hour_used_pct REAL,
  five_hour_resets_at TEXT,
  seven_day_used_pct REAL,
  seven_day_resets_at TEXT
);
```

### 5.2 Transcript Events

Parsed from `~/.claude/sessions/{id}.jsonl` and stored in `transcript_events`:

```sql
CREATE TABLE transcript_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp TEXT NOT NULL,
  session_id TEXT NOT NULL,
  event_type TEXT NOT NULL,         -- tool_call | skill | agent | todo
  name TEXT,                        -- tool/skill/agent name
  status TEXT,                      -- active | done | queued
  detail TEXT,                      -- e.g. file path, agent model
  duration_ms INTEGER,
  todo_total INTEGER,
  todo_done INTEGER
);
```

### 5.3 Skills & Plugins Registry

Scraped once at startup from `~/.claude/settings.json` + `~/.claude/plugins/`. Cached in memory — doesn't need SQLite.

---

## 6. Component Tree (Bubble Tea Model)

```
App
├── TopBar
│   ├── ProjectName
│   ├── GitBranch
│   └── ThemeIndicator
│
├── Viewport (flex layout, left 70% / right 30%)
│   ├── AnimationArea (theme-dependent)
│   │   ├── SpaceshipView
│   │   │   ├── ParticleField (stars)
│   │   │   ├── SpeedometerGauge
│   │   │   └── WarpSpeedLine
│   │   └── DJView
│   │       ├── SpectrumBar (input)
│   │       ├── BreathingHalo
│   │       └── SpectrumBar (output)
│   │
│   └── InfoPanel (right column)
│       ├── MetricsCard
│       ├── ActivityCard
│       ├── AgentsCard
│       └── TodosCard
│
├── Sidebar (conditionally rendered, `s` key)
│   ├── SkillsList
│   ├── PluginsList
│   └── Recommendation
│
├── HelpOverlay (conditionally rendered, `?` key)
└── StatusBar (key hints)
```

---

## 7. Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `t` | Toggle theme (spaceship ↔ DJ) |
| `s` | Toggle skills/plugins sidebar |
| `?` | Toggle help overlay |
| `d` | Toggle drift chat (Phase 2) |
| `q` / `Ctrl+C` | Quit |
| `r` | Force refresh all data |

---

## 8. Development Phases

### Phase 0 — Wiring (1 afternoon, no code written)
- Verify statusline pipe works: ensure `vibeship collect` receives stdin JSON
- Verify transcript JSONL is readable at `~/.claude/sessions/`
- Run `ccusage` once to confirm data access

### Phase 1 — MVP (Both Pillars)
1. **Project scaffold** — Go module init, Bubble Tea app shell, basic rendering loop
2. **Data ingest** — `vibeship collect` (stdin JSON → SQLite), transcript polling goroutine
3. **Metrics card** — Cost, context %, rate limit from SQLite
4. **Activity card** — Active tool/skill/MCP from transcript events
5. **Agents card** — Running agents from transcript events
6. **Todos card** — Todo progress from transcript events
7. **Spaceship animation** — Particle field + speedometer gauge + speed line
8. **DJ animation** — Spectrum bars + breathing halo
9. **Theme switch** — `t` key, smooth transition
10. **Skills sidebar** — `s` key, all skills/plugins + recommendation
11. **Thinking co-pilot** — `d` key overlay, SCOPE.md parsing, auto check questions, Q&A logging
12. **Packaging** — `go install`, Homebrew formula

### Phase 2 — Control Layer (Hooks)
1. **Budget guard** — PreToolUse hook, exit 2 on threshold breach
2. **Enhanced scope drift** — PreToolUse hook diffs Write/Edit against SCOPE.md, blocks out-of-scope
3. **Why-trail** — Decision log enriched with hook context
4. **Rollback anchors** — Git milestone integration

### Phase 3 — Polish
1. **Custom themes** — User-configurable color palettes
2. **Multi-profile support** — `CLAUDE_CONFIG_DIR` awareness
3. **Web dashboard** — Optional browser-based view (long-term, not committed)

---

## 9. Error & Edge Case Handling

| Scenario | Behavior |
|----------|----------|
| Claude Code not running | Vibeship shows "Waiting for Claude Code…" with last known data or empty state |
| No session data yet (fresh install) | Show empty dashboard with onboarding hint: "Start Claude Code to see data" |
| Stale data (>30s no statusline update) | Dim the metrics card, show "stale" indicator |
| Transcript JSONL missing or unparseable | Show "—" for activity/agents/todos cards, don't crash |
| SQLite locked | Retry 3x with exponential backoff, show "busy" if persistent |
| Terminal too small (<80×24) | Show compact single-column mode with reduced animation |
| Binary not in PATH | Display setup instructions on first run |

---

## 10. Non-Goals (Explicitly Out of Scope for MVP)

- LLM-powered recommendations (uses simple rules engine instead)
- Multi-user support
- Cloud sync / team dashboards
- Integration with non-Claude Code agents
- Web-based dashboard
- Historical analytics beyond current session

---

## 11. Open Questions (Defer to Implementation)

- Exact Bubble Tea animation frame rate (start at 15fps, tune)
- SQLite WAL mode vs standard journal
- Whether `vibeship collect` should be a separate binary or `vibeship collect` subcommand
- Exact particle count for spaceship theme (start at 50, tune for performance)
