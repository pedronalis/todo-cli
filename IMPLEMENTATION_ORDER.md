# IMPLEMENTATION_ORDER.md — Fases e dependências

## Visão geral
Objetivo: reduzir risco, entregar valor cedo e manter implementação cirúrgica por módulos.

Ordem macro: **Domínio/Store → App Controller → TUI mínima → Features MVP+ → Robustez final**.

---

## Fase 0 — Setup e contratos congelados
**Entregas**
- Confirmar contratos deste documento + `SPEC.md` + `ARCHITECTURE.md`.
- Definir checklist de DoD e cenários de teste manuais.

**Dependências**
- Nenhuma.

**Critério de saída**
- Time alinhado em nomes de intents, modelo JSON e keymap.

---

## Fase 1 — Núcleo de domínio (sem TUI)
**Entregas**
- Tipos de `List`, `Task`, `AppState`, `Filter`.
- Reducers puros para CRUD de listas/tarefas, toggle, prioridade, ordenação.
- Selectors para tarefas filtradas e contadores.
- Validações e invariantes.

**Dependências**
- Fase 0.

**Critério de saída**
- Testes de unidade do domínio passando.
- Mutação de estado previsível e determinística.

---

## Fase 2 — Store (persistência, autosave, backup)
**Entregas**
- `load/save` JSON com schema versionado.
- Escrita atômica (`tmp + rename`).
- Política de backup rotativo.
- Recuperação automática em corrupção.

**Dependências**
- Fase 1 (tipos e estado).

**Critério de saída**
- Reinício preserva estado.
- Simulação de JSON inválido recupera backup corretamente.

---

## Fase 3 — App Controller + Undo
**Entregas**
- Dispatcher de intents.
- Orquestração domínio + store.
- Undo stack (mínimo 20 ações).
- Eventos de status para UI.

**Dependências**
- Fase 1 e Fase 2.

**Critério de saída**
- Fluxos de criação/edição/exclusão/toggle/move/pioridade/undo funcionando sem TUI (via testes ou harness).

---

## Fase 4 — TUI base (MVP Core)
**Entregas**
- Layout 2 painéis (Listas/Tarefas) + status bar.
- Navegação por teclado (`Tab`, `j/k`, `Enter`, `q`).
- Quick-add de lista/tarefa (`a`) com prompt simples.
- Ações core: criar, editar, excluir, concluir.

**Dependências**
- Fase 3.

**Critério de saída**
- Core do MVP utilizável de ponta a ponta no terminal.

---

## Fase 5 — Recursos MVP+ (incremental)
**Entregas**
1. Filtro textual (`/`) e filtro status (`f`).
2. Undo exposto em UI (`u`).
3. Prioridade (`1..4`) e ordenação (`J/K`).
4. Confirmações de deleção e mensagens de status claras.

**Dependências**
- Fase 4.

**Critério de saída**
- Core + 5 melhorias do SPEC disponíveis no fluxo real.

---

## Fase 6 — Hardening e pronto para handoff
**Entregas**
- Ajustes de UX (mensagens, foco, comportamento de `Esc`).
- Testes de regressão manuais guiados por DoD.
- Checklist final de desempenho e robustez.

**Dependências**
- Fase 5.

**Critério de saída**
- Todos critérios de pronto (DoD) atendidos.
- Documentação final consistente com comportamento implementado.

---

## Matriz de dependências (resumo)
- **F0** → base para todas.
- **F1** → pré-requisito de F2 e F3.
- **F2 + F1** → pré-requisito de F3.
- **F3** → pré-requisito de F4.
- **F4** → pré-requisito de F5.
- **F5** → pré-requisito de F6.

---

## Estratégia de paralelização segura
- Enquanto 1 dev implementa `store` (F2), outro pode preparar esqueleto de `tui` sem lógica final.
- Integração oficial só após contratos de intents congelados em F3.
- Evitar trabalho paralelo em reducers sem suíte de testes compartilhada.
