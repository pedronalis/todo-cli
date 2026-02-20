# SPEC.md — ToDo CLI Minimalista (MVP+)

## 1) Objetivo do produto
Um app de tarefas em terminal, rápido e sem distrações, focado em:
- capturar tarefas em segundos,
- organizar por listas,
- executar com navegação por teclado,
- manter dados locais confiáveis.

---

## 2) Escopo MVP+ final

## 2.1 Core (MVP)
1. **Múltiplas listas** (criar, renomear, excluir).
2. **CRUD de tarefas** dentro da lista selecionada.
3. **Marcar/desmarcar concluída**.
4. **Navegação 100% por teclado** com foco alternável entre painéis.
5. **Persistência local em JSON** (carregar ao iniciar, salvar ao alterar).

## 2.2 +5 melhorias (MVP+)
1. **Filtro de tarefas** por status (`todas`, `abertas`, `concluídas`) + busca textual.
2. **Quick-add contextual** (adicionar lista/tarefa sem sair do fluxo).
3. **Undo de última ação** (pilha curta, ex.: até 20 ações).
4. **Prioridade e ordenação manual de tarefas** (subir/descer).
5. **Autosave com backup rotativo e recuperação** em caso de corrupção.

---

## 3) Fluxos de uso

## 3.1 Fluxo: listas
1. Usuário abre app.
2. Painel **Listas** mostra coleções (ex.: Inbox, Trabalho, Pessoal).
3. Usuário navega (`j/k` ou setas).
4. Usuário cria nova lista (`a` com foco em listas).
5. Usuário renomeia (`r`) ou exclui (`d`, com confirmação).
6. Ao selecionar lista (`Enter`), painel de tarefas é atualizado.

## 3.2 Fluxo: tarefas
1. Usuário muda foco para **Tarefas** (`Tab`).
2. Navega tarefas (`j/k`).
3. Adiciona tarefa (`a` com foco em tarefas, quick-add).
4. Marca concluída (`x`) e desfaz se necessário (`u`).
5. Edita título (`e`) ou exclui (`d`, com confirmação).
6. Ajusta prioridade (`1..4`) e reordena (`J/K`).

## 3.3 Fluxo: filtro
1. Usuário abre filtro (`/`).
2. Digita termo (busca incremental).
3. Alterna status (`f`) entre abertas/concluídas/todas.
4. Limpa filtro (`Esc`).

## 3.4 Fluxo: undo
1. Usuário executa ação mutável (criar, editar, mover, concluir, excluir).
2. App registra snapshot/patch no stack de undo.
3. Usuário pressiona `u` para desfazer última ação.
4. Estado volta ao anterior e salva automaticamente.

---

## 4) Keymap final

## 4.1 Globais
- `q`: sair (com flush de autosave pendente)
- `Tab`: alternar foco entre **listas** e **tarefas**
- `u`: undo da última ação
- `/`: abrir busca textual
- `Esc`: fechar prompt/filtro e voltar ao modo normal

## 4.2 Com foco em Listas
- `j` / `k` (ou `↓` / `↑`): navegar listas
- `Enter`: selecionar lista
- `a`: quick-add de lista
- `r`: renomear lista selecionada
- `d`: excluir lista selecionada (com confirmação)

## 4.3 Com foco em Tarefas
- `j` / `k` (ou `↓` / `↑`): navegar tarefas
- `a`: quick-add de tarefa na lista ativa
- `x`: alternar concluída/aberta
- `e`: editar título
- `d`: excluir tarefa (com confirmação)
- `J` / `K`: mover tarefa para baixo/cima
- `1`: prioridade nenhuma
- `2`: prioridade baixa
- `3`: prioridade média
- `4`: prioridade alta
- `f`: alternar filtro de status (todas → abertas → concluídas)

---

## 5) Máquina de estados da UI

## 5.1 Estados principais
- **NORMAL**
  - subestado `focusPane`: `LISTS | TASKS`
- **FILTER_INPUT**
  - edição de busca textual
- **QUICK_ADD_LIST**
- **QUICK_ADD_TASK**
- **CONFIRM_DELETE**

## 5.2 Eventos e transições (resumo)
- `Tab` em `NORMAL` → alterna `focusPane`.
- `/` em `NORMAL` → `FILTER_INPUT`.
- `a` em `NORMAL + LISTS` → `QUICK_ADD_LIST`.
- `a` em `NORMAL + TASKS` → `QUICK_ADD_TASK`.
- `d` em `NORMAL` → `CONFIRM_DELETE`.
- `Enter` em estados de input/confirmação → aplica ação, salva, retorna a `NORMAL`.
- `Esc` em qualquer estado modal → cancela e retorna a `NORMAL`.
- `u` em `NORMAL` → aplica undo (se stack não vazio), permanece em `NORMAL`.

## 5.3 Regras de consistência
- Filtro ativo **não** altera dados, só visão.
- Undo atua sobre o estado de dados (listas/tarefas/metadados), não sobre foco visual.
- Se lista/tarefa selecionada deixar de existir após undo/delete, seleção cai para item vizinho válido.

---

## 6) Critérios de pronto (Definition of Done)

## 6.1 Funcional
- Todas ações do Core + 5 melhorias operam por teclado.
- CRUD de listas e tarefas com confirmação de deleção.
- Filtro textual + status funcionando combinados.
- Undo reverte corretamente ações suportadas (mínimo: criar, editar, concluir, mover, excluir).
- Ordenação manual e prioridade persistem após reinício.

## 6.2 Persistência e robustez
- Estado salvo em JSON válido após cada mutação (autosave).
- Escrita atômica sem corrupção em interrupção simples.
- Backup rotativo criado conforme regra definida.
- Em JSON corrompido, app recupera de backup e informa usuário.

## 6.3 UX e desempenho
- Inicialização percebida < 300ms em dataset pequeno/médio local.
- Navegação sem flicker perceptível em uso normal.
- Mensagens de status claras (ex.: “Tarefa criada”, “Undo aplicado”).

## 6.4 Qualidade
- Contratos entre `app`, `store`, `tui` respeitados.
- Logs de erro mínimos e acionáveis.
- Documentação alinhada com `ARCHITECTURE.md` e `IMPLEMENTATION_ORDER.md`.
