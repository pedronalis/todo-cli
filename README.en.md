# ✅ todo-cli

> A keyboard-first terminal ToDo app built with Go + Bubble Tea.

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)
![Bubble Tea](https://img.shields.io/badge/TUI-Bubble%20Tea-FF5F87)
![License](https://img.shields.io/badge/license-MIT-green)

![App screenshot](docs/images/todo-cli-main.jpg)

---

## ✨ Overview

**todo-cli** is designed for fast terminal workflows:

- 🗂️ multiple lists
- ✅ open/completed tasks
- 🎯 per-task priority
- 🔎 incremental search + status filters
- 🧠 undo support
- 🧾 archive/history for completed tasks
- 💾 local persistence with automatic backup/recovery

---

## 🧱 Architecture (representation)

```mermaid
flowchart LR
  UI[TUI - Bubble Tea/Lipgloss] --> APP[Business rules (app)]
  APP --> MODEL[Domain model (model)]
  APP --> STORE[JSON persistence (store)]
  STORE --> FILE[(state.json + backups)]
```

### Project structure

- `cmd/todo` → entrypoint
- `tui` → terminal UI
- `app` → business logic
- `model` → domain types
- `store` → load/save/autosave/recovery
- `docs/images` → screenshots/assets

---

## 🚀 Run locally

```bash
go run ./cmd/todo
```

With custom state path:

```bash
go run ./cmd/todo -state /path/to/state.json
```

Build:

```bash
go build -o todo ./cmd/todo
./todo
```

---

## ⌨️ Core keymap

| Context | Action | Key |
|---|---|---|
| Global | Switch focus | `Tab` |
| Global | Navigate | `j/k` or `↑/↓` |
| Global | Incremental search | `/` |
| Global | Undo | `u` |
| Global | Help | `?` |
| Global | Quit | `q` |
| Lists | Add list | `a` |
| Lists | Rename | `r` |
| Lists | Reorder | `J/K` |
| Tasks | Add task | `a` |
| Tasks | Toggle done | `x` |
| Tasks | Edit | `e` |
| Tasks | Priority | `1..4` |
| Tasks | Filter | `f` |
| Tasks | Archive completed | `C` |
| Tasks | Archive all | `A` |
| Tasks | Delete all | `D` |
| Tasks | Copy active tasks | `y` |

---

## 🛡️ Persistence & reliability

- Autosaves after relevant mutations
- Keeps `.bak` + rotating snapshot backups
- Tries automatic recovery if `state.json` is corrupted

---

## 🧪 Tests

```bash
go test ./...
go build ./...
```

---

## 📄 License

MIT
