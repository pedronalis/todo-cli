# ✅ todo-cli

> Um ToDo de terminal rápido, bonito e sem firula — feito em **Go + Bubble Tea**.

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)
[![Bubble Tea](https://img.shields.io/badge/TUI-Bubble%20Tea-FF5F87)](https://github.com/charmbracelet/bubbletea)
[![License](https://img.shields.io/badge/license-MIT-green)](#-licença)

---

## ✨ O que ele faz

- 🗂️ Múltiplas listas (com cor e reordenação)
- ✅ Tarefas com status aberto/concluído
- 🎯 Prioridade por tarefa (`1..4`)
- 🔎 Busca incremental + filtro (`todas/abertas/concluídas`)
- 🧠 Undo (`u`) para desfazer ações
- 🧾 Histórico de concluídas por lista
- 📋 Copiar to-dos ativos para clipboard (`y`)
- 💾 Autosave com backup e recovery de JSON corrompido

---

## 🧰 Stack

- [Go](https://go.dev/)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lipgloss](https://github.com/charmbracelet/lipgloss)

---

## 🚀 Rodando local

```bash
go run ./cmd/todo
```

Com arquivo de estado customizado:

```bash
go run ./cmd/todo -state /caminho/estado.json
```

Build:

```bash
go build -o todo ./cmd/todo
./todo
```

---

## ⌨️ Keymap essencial

### Global

- `Tab`: alterna foco entre listas/tarefas
- `j/k` ou `↑/↓`: navega
- `/`: busca incremental
- `u`: desfaz última ação
- `?`: abre/fecha atalhos
- `q`: sair

### Listas

- `a`: criar lista
- `r`: renomear
- `c`: trocar cor
- `J/K`: reordenar
- `d`: excluir (com confirmação)

### Tarefas

- `a`: criar tarefa
- `e`: editar tarefa
- `x`: concluir/reabrir
- `1..4`: prioridade
- `f`: alterna filtro
- `y`: copiar to-dos ativos
- `C`: arquivar concluídas
- `A`: arquivar todas
- `D`: deletar todas

---

## 🏗️ Estrutura do projeto

- `cmd/todo` → entrypoint
- `model` → tipos de domínio
- `app` → regras de negócio
- `store` → persistência/autosave/recovery
- `tui` → interface terminal

---

## 🧪 Testes

```bash
go test ./...
go build ./...
```

---

## 🗺️ Roadmap (idéias)

- [ ] Exportar markdown por lista
- [ ] Tema claro/escuro configurável
- [ ] Filtro por prioridade
- [ ] Sync opcional (Git / cloud)

---

## 📄 Licença

MIT
