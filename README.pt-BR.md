# ✅ todo-cli

> Um ToDo de terminal **rápido, bonito e sem fricção**, feito com Go + Bubble Tea.

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)
![Bubble Tea](https://img.shields.io/badge/TUI-Bubble%20Tea-FF5F87)
![Licença](https://img.shields.io/badge/licença-MIT-green)

![Screenshot do app](docs/images/todo-cli-main.jpg)

---

## ✨ Visão geral

O **todo-cli** é um app de produtividade focado em fluxo de teclado:

- 🗂️ múltiplas listas
- ✅ tarefas abertas/concluídas
- 🎯 prioridade por tarefa
- 🔎 busca incremental e filtros
- 🧠 undo confiável
- 🧾 histórico de concluídas
- 💾 persistência local com backup automático

---

## 🧱 Arquitetura (representação)

```mermaid
flowchart LR
  UI[TUI - Bubble Tea/Lipgloss] --> APP[Camada de regras (app)]
  APP --> MODEL[Tipos de domínio (model)]
  APP --> STORE[Persistência JSON (store)]
  STORE --> FILE[(state.json + backups)]
```

### Estrutura de pastas

- `cmd/todo` → entrypoint
- `tui` → interface terminal
- `app` → regras de negócio
- `model` → entidades e tipos
- `store` → load/save/autosave/recovery
- `docs/images` → screenshots/imagens

---

## 🚀 Como rodar

```bash
go run ./cmd/todo
```

Com estado customizado:

```bash
go run ./cmd/todo -state /caminho/estado.json
```

Build:

```bash
go build -o todo ./cmd/todo
./todo
```

---

## ⌨️ Atalhos principais

| Contexto | Ação | Tecla |
|---|---|---|
| Global | Alternar foco | `Tab` |
| Global | Navegar | `j/k` ou `↑/↓` |
| Global | Busca incremental | `/` |
| Global | Desfazer | `u` |
| Global | Ajuda | `?` |
| Global | Sair | `q` |
| Listas | Criar lista | `a` |
| Listas | Renomear | `r` |
| Listas | Reordenar | `J/K` |
| Tarefas | Criar tarefa | `a` |
| Tarefas | Concluir/Reabrir | `x` |
| Tarefas | Editar | `e` |
| Tarefas | Prioridade | `1..4` |
| Tarefas | Filtro | `f` |
| Tarefas | Arquivar concluídas | `C` |
| Tarefas | Arquivar todas | `A` |
| Tarefas | Deletar todas | `D` |
| Tarefas | Copiar ativas | `y` |

---

## 🛡️ Persistência e robustez

- Salva automaticamente a cada mutação relevante
- Cria backup (`.bak`) + snapshots rotativos
- Se `state.json` corromper, tenta recuperação automática

---

## 🧪 Testes

```bash
go test ./...
go build ./...
```

---

## 🗺️ Roadmap curto

- [ ] Exportar markdown por lista
- [ ] Filtro por prioridade
- [ ] Tema configurável
- [ ] GIF de demonstração no README

---

## 📄 Licença

MIT
