# USABILITY_PLUS_REPORT.md

## Escopo do pass de usabilidade

Projeto avaliado: `/home/pedro/.openclaw/workspace/projects/todo-cli`

Este pass implementa as melhorias pedidas para MVP+ mantendo minimalismo, teclado-first e compatibilidade com estados antigos.

---

## Mudanças implementadas

### 1) Reordenação robusta de listas e tarefas

#### Tarefas
- Mantido `J/K` para mover tarefa para baixo/cima.
- Mensagens robustas de limite:
  - "A tarefa já está no topo"
  - "A tarefa já está no fim"
- Persistência via autosave após reorder.

#### Listas (novo)
- Implementado reorder de listas também com `J/K` quando o foco está em **Listas**.
- Novos erros de domínio:
  - `ErrListAlreadyAtTop`
  - `ErrListAlreadyAtBottom`
- Persistência imediata do novo ordenamento.

Decisão UX: manter **consistência de atalhos** (`J/K` sem multiplicar teclas), mudando o alvo conforme foco atual.

---

### 2) Prioridade por cores (substituindo visual "!!!")

- Prioridade agora é renderizada por **indicador colorido discreto** (`●`) antes do texto:
  - baixa: verde
  - média: amarelo
  - alta: vermelho
  - nenhuma: ponto neutro/faint
- Remove dependência visual de "!!!".

Decisão UX: manter leitura rápida e baixo ruído visual.

---

### 3) Acúmulo de concluídas → arquivamento com histórico persistente (NOVA DIRETRIZ)

Implementado fluxo de arquivamento sem perda de histórico.

#### Novo modelo persistente
Adicionado no estado: `archivedCompleted`.

Cada item arquivado guarda:
- `taskText`
- `originList`
- `originListId`
- `priority`
- `doneAt` (timestamp da conclusão)
- `archivedAt` (timestamp do arquivamento)

#### Nova ação principal
- Tecla `C` (com foco em tarefas):
  - abre confirmação para arquivar concluídas da lista ativa
  - move concluídas para histórico (não descarta)
  - remove concluídas da lista ativa
  - mantém undo disponível

#### Undo
- `u` desfaz o arquivamento completo (restaura tarefas ativas e remove entradas recém-arquivadas).

#### Visualização de histórico
- Tecla `h` abre/fecha painel de **Histórico de concluídas** (somente leitura).
- Histórico mostra texto, lista de origem, prioridade e timestamp.

Decisão UX (item crítico):
- Em vez de "delete concluídas", adotado **"arquivar concluídas"**.
- Isso resolve poluição da lista ativa preservando trilha histórica.
- Undo continua funcional e previsível.

---

### 4) Melhorias gerais de usabilidade

#### Hint contextual no rodapé
- Rodapé agora muda por foco/modo:
  - listas
  - tarefas
  - histórico
  - busca
  - confirmações

#### Onboarding de primeiro uso
- Exibição de onboarding quando:
  - `metadata.firstRun == true` **ou**
  - não há listas/tarefas.
- `firstRun` é marcado como concluído após primeira persistência de ação útil.

#### Busca incremental
- Modo `/` agora aplica filtro **enquanto digita** (não só no Enter).
- `Enter` confirma e persiste.
- `Esc` limpa busca.

#### Persistência de contexto da sessão
Persistido em `metadata.session`:
- `activeListId`
- `focus` (`lists`/`tasks`)

Também permanecem persistidos filtro e query (já existentes no estado).

#### Toast curto pós-ação
- Status pós-ação foi padronizado com mensagens curtas.
- Quando aplicável, inclui dica de undo (`• u desfaz`).

---

## Compatibilidade com estado antigo

Compatibilidade preservada:
- Estados legados sem `archivedCompleted`, `metadata.session` ou `firstRun` continuam carregando.
- Defaults são aplicados sem quebrar leitura.
- Prioridades inválidas legadas seguem normalizadas.

---

## Arquivos alterados

- `model/model.go`
- `model/model_test.go`
- `app/app.go`
- `app/app_test.go`
- `store/store.go`
- `store/store_test.go`
- `tui/tui.go`
- `README.md`
- `USABILITY_PLUS_REPORT.md` (novo)

---

## Validação obrigatória

### Testes
Executado:

```bash
go test ./...
```

Resultado:

- `ok   todo-cli/app`
- `ok   todo-cli/model`
- `ok   todo-cli/store`
- `?    todo-cli/cmd/todo [no test files]`
- `?    todo-cli/tui [no test files]`

### Build
Executado:

```bash
go build ./...
```

Resultado: sem erros.

---

## Smoke test manual (teclado, cenários críticos)

Foi executado smoke manual em TUI via sessão PTY (`go run ./cmd/todo -state ./smoke-usability-plus-2.json`).

### Cenário A — fluxo básico + arquivamento + histórico + undo
1. Criar lista (`a` em Listas)
2. Tab para Tarefas
3. Criar tarefa (`a`)
4. Marcar concluída (`x`)
5. Arquivar concluídas (`C`, confirmar `y`)
6. Abrir histórico (`h`) e validar item exibido
7. Undo (`u`) para desfazer arquivamento
8. Sair (`q`)

Observações:
- Histórico abriu com item arquivado visível.
- Undo restaurou tarefa concluída para lista ativa.
- Sessão encerrou com exit code 0.

### Cenário B — persistência de contexto
Arquivo de estado pós-smoke (`smoke-usability-plus-2.json`) confirmou persistência de:
- `metadata.session.activeListId`
- `metadata.session.focus`

---

## Limitações conhecidas (aceitas para MVP+)

1. Histórico é somente leitura (não há busca/filtro dedicado dentro dele).
2. `doneAt` usa o timestamp de última atualização da tarefa (boa aproximação no modelo atual).
3. Feedback de "toast" usa a linha de status (não há timer visual dedicado para auto-expirar).

Esses pontos mantêm o app simples e coerente com o objetivo minimalista atual.
