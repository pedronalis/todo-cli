# ARCHITECTURE.md — ToDo CLI Minimalista

## 1) Princípios arquiteturais
- **Minimalismo**: poucos módulos, responsabilidades claras.
- **Separação forte**: UI (TUI) não manipula arquivo diretamente.
- **Estado previsível**: mutações passam por camada `app`.
- **Confiabilidade local-first**: autosave + backup + recuperação.

---

## 2) Estrutura de pastas/arquivos (proposta)

```text
todo-cli/
  SPEC.md
  ARCHITECTURE.md
  IMPLEMENTATION_ORDER.md
  src/
    app/
      index.ts            # bootstrap + loop principal de intents
      controller.ts       # orquestra intents -> comandos de domínio
      commands.ts         # comandos mutáveis e não mutáveis
      undo.ts             # gestão de stack de undo
    domain/
      types.ts            # tipos de List, Task, AppState, filtros
      validators.ts       # validações de entrada e invariantes
      selectors.ts        # consultas derivadas (tarefas filtradas etc.)
      reducers.ts         # mutações puras de estado
    store/
      fileStore.ts        # load/save atômico
      backup.ts           # rotação e restauração de backups
      paths.ts            # resolução de caminhos (data/backup)
      schema.ts           # versão de schema + migração simples
    tui/
      renderer.ts         # desenha painéis, barra de status, prompts
      keymap.ts           # mapeamento de teclas -> intents
      prompts.ts          # quick-add, confirmação, filtro
      state.ts            # estado visual (foco, modal, cursor)
    shared/
      time.ts
      logger.ts
      result.ts
  data/
    todo.json             # opcional em dev local
  backups/
    *.json
```

> Observação: nomes de arquivo são sugestão de contrato estrutural; a implementação pode variar mantendo as fronteiras.

---

## 3) Modelo de dados JSON

## 3.1 Estrutura
```json
{
  "version": 1,
  "meta": {
    "createdAt": "2026-02-19T12:00:00.000Z",
    "updatedAt": "2026-02-19T12:05:00.000Z",
    "nextIds": { "list": 3, "task": 8 }
  },
  "lists": [
    {
      "id": "l1",
      "name": "Inbox",
      "createdAt": "2026-02-19T12:00:00.000Z",
      "updatedAt": "2026-02-19T12:00:00.000Z",
      "archived": false,
      "taskOrder": ["t1", "t2"]
    }
  ],
  "tasks": [
    {
      "id": "t1",
      "listId": "l1",
      "title": "Comprar café",
      "done": false,
      "priority": "none",
      "createdAt": "2026-02-19T12:01:00.000Z",
      "updatedAt": "2026-02-19T12:01:00.000Z",
      "completedAt": null
    }
  ],
  "uiPrefs": {
    "selectedListId": "l1",
    "focusPane": "TASKS",
    "filter": {
      "status": "open",
      "query": ""
    }
  }
}
```

## 3.2 Regras/invariantes
- `id` é único por coleção (`lists`, `tasks`).
- Toda `task.listId` deve apontar para lista existente.
- `taskOrder` contém apenas tarefas da própria lista.
- `priority` ∈ `none | low | medium | high`.
- `completedAt` só existe quando `done = true`.
- `version` obrigatório para migração futura.

---

## 4) Contratos de funções por módulo

## 4.1 Contrato TUI → APP (intents)
A TUI **não** altera estado direto; envia intents:
- `NAVIGATE_LIST(delta)`
- `NAVIGATE_TASK(delta)`
- `SWITCH_FOCUS()`
- `CREATE_LIST(name)`
- `RENAME_LIST(listId, name)`
- `DELETE_LIST(listId)`
- `CREATE_TASK(listId, title)`
- `EDIT_TASK(taskId, title)`
- `TOGGLE_TASK(taskId)`
- `DELETE_TASK(taskId)`
- `MOVE_TASK(taskId, direction)`
- `SET_PRIORITY(taskId, priority)`
- `SET_FILTER_STATUS(status)`
- `SET_FILTER_QUERY(query)`
- `UNDO()`
- `QUIT()`

## 4.2 Módulo `app` (orquestração)
Responsável por interpretar intent, chamar reducers/store e devolver estado para render.

Funções contrato (conceitual):
- `bootstrap(): Promise<AppState>`
- `dispatch(intent: Intent): Promise<AppState>`
- `getState(): AppState`
- `subscribe(listener: (state) => void): Unsubscribe`
- `shutdown(): Promise<void>`

Responsabilidades:
- validação de intent,
- aplicação de mutações (via domínio),
- gravação no store,
- controle de undo stack,
- publicação de mensagens de status para a TUI.

## 4.3 Módulo `store` (persistência)
Funções contrato (conceitual):
- `load(): Promise<AppState>`
- `save(state: AppState, reason: SaveReason): Promise<void>`
- `saveAtomic(state): Promise<void>`
- `createBackup(state, tag?): Promise<string>`
- `listBackups(): Promise<BackupInfo[]>`
- `restoreLatestValidBackup(): Promise<AppState>`
- `validateSchema(raw): ValidationResult`
- `migrateIfNeeded(raw): AppState`

Responsabilidades:
- serialização/deserialização JSON,
- escrita atômica (temp + rename),
- política de backups,
- recuperação de falhas.

## 4.4 Módulo `tui` (apresentação)
Funções contrato (conceitual):
- `init(onIntent: (intent) => void): void`
- `render(state: AppState, ui: UIState): void`
- `showPrompt(type, initial?): Promise<string | null>`
- `showConfirm(message): Promise<boolean>`
- `setStatus(message, level?): void`
- `destroy(): void`

Responsabilidades:
- desenhar interface,
- coletar input de teclado,
- traduzir keymap para intents,
- não conhecer detalhes de arquivo/backup.

---

## 5) Regras de autosave e backup

## 5.1 Autosave
- Disparar em toda mutação de dados.
- Debounce de **300–500ms** para agrupar ações rápidas.
- `QUIT` força flush imediato.
- Em erro de save: manter app aberto, exibir erro e permitir retry automático na próxima mutação.

## 5.2 Escrita atômica
1. Serializa estado para `todo.json.tmp`.
2. `fsync` do arquivo temporário (quando disponível).
3. `rename` para `todo.json` (operação atômica no mesmo filesystem).

## 5.3 Política de backup
- Criar backup:
  - a cada **20 saves** ou
  - a cada **10 minutos** desde último backup,
  - o que ocorrer primeiro.
- Nome: `backup-YYYYMMDD-HHmmss.json`.
- Manter últimos **10** backups válidos (rotação FIFO por timestamp).

## 5.4 Recuperação
- Ao iniciar, se `todo.json` inválido/corrompido:
  1. mover arquivo ruim para `todo.corrupt-<timestamp>.json`;
  2. restaurar backup mais recente válido;
  3. informar no status/log.
- Se não houver backup válido: iniciar estado vazio padrão e alertar explicitamente.

---

## 6) Limites do escopo arquitetural
- Sem sync cloud no MVP+.
- Sem múltiplos usuários.
- Sem edição longa de notas ricas (apenas título de tarefa no MVP+).
- Sem plugins.
