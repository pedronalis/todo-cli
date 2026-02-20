# POLISHING_REPORT.md

## Resumo executivo

Fase de polishing concluída com foco em **confiabilidade**, depois UX e visual minimalista.

Resultado: o app ficou mais aderente ao `SPEC.md` (MVP+), com atalhos e fluxos importantes fechados, mensagens mais claras, robustez melhor em persistência (backup rotativo + recuperação de corrupção) e documentação atualizada para uso diário local.

---

## 1) Auditoria de qualidade (SPEC vs implementação)

### Gaps encontrados (antes do polishing)

1. **Atalhos MVP+ faltando na TUI**
   - `r` (renomear lista)
   - `e` (editar tarefa)
   - `J/K` (reordenação manual)
   - `1..4` (prioridade)

2. **Recuperação de JSON corrompido ausente no startup**
   - `store.Load` falhava e encerrava o app sem fallback.

3. **Mensagens/fluxos em estados vazios**
   - Alguns atalhos em foco errado eram silenciosos ou pouco claros.
   - Estados vazios podiam ficar ambíguos (lista vazia vs filtro/busca sem resultado).

4. **UX de input com espaço**
   - Espaço não era capturado em modos de input (`nova tarefa`, `renomear`, `editar`).

### Situação após polishing

- Core + melhorias MVP+ relevantes no `SPEC.md`: **cobertas no fluxo principal**.
- Persistência robusta com recuperação automática de corrupção: **implementada**.
- Keymap e README: **alinhados com o comportamento atual**.

---

## 2) Polimento funcional aplicado

### 2.1 Atalhos/fluxo

Implementado na TUI:
- `r` (listas): renomear lista selecionada
- `e` (tarefas): editar tarefa selecionada
- `J/K` (tarefas): mover para baixo/cima
- `1..4` (tarefas): prioridade nenhuma/baixa/média/alta
- `Enter` com foco em listas: confirma lista ativa e status explícito

Também foi reforçado:
- validação de foco (atalho de tarefa só funciona no painel de tarefas e vice-versa)
- mensagens orientativas quando foco está incorreto

### 2.2 Mensagens de erro/status

Melhorias principais:
- mensagens com contexto de ação (ex.: `Erro ao editar tarefa`, `Erro ao renomear lista`)
- status específico para limites de ordenação (`já está no topo/fim`)
- feedback explícito em cancelamentos e estados sem seleção
- mensagem de persistência mais honesta em falha de I/O:
  - `Alteração aplicada, mas falhou ao salvar em disco: ...`

### 2.3 Estados vazios

A visão de tarefas agora diferencia:
- nenhuma lista ativa
- lista ativa sem tarefas
- tarefas ocultas por filtro/busca

### 2.4 Autosave/backup e I/O

No `store`:
- `Autosave` continua atômico (temp + rename)
- backup agora inclui:
  - `state.json.bak` (último snapshot)
  - `state.json.bak.<timestamp>` (rotativo, máx. 10)
- prune automático dos backups antigos

Recuperação:
- nova rotina `LoadWithRecovery`
- se JSON principal estiver corrompido:
  1. move arquivo ruim para `state.corrupt-<timestamp>.json`
  2. tenta restaurar do backup válido mais recente
  3. se não houver backup válido, inicia estado vazio e salva
  4. retorna mensagem de status para informar o usuário

---

## 3) Polimento visual minimalista

Sem overengineering, mantendo terminal-friendly:
- títulos de painel com indicação de foco ativo (`[ativo]`)
- resumo superior mais legível (`foco`, `filtro`, `busca`)
- destaque visual consistente de seleção
- tarefa concluída com visual suave (faint + strikethrough)
- textos de ajuda/status mais diretos

---

## 4) Bugs corrigidos

1. **Espaço não funcionava nos inputs**
   - Causa: `tea.KeySpace` não era tratado.
   - Fix: suporte explícito a `KeySpace` em `updateInputMode`.

2. **MVP+ incompleto em atalhos da TUI**
   - Causa: service já tinha APIs, mas TUI não expunha todos os fluxos.
   - Fix: integração dos atalhos `r/e/J/K/1..4`.

3. **Sem recuperação de corrupção no startup**
   - Causa: caminho único de load sem fallback.
   - Fix: `LoadWithRecovery` + restauração por backup + arquivo corrupt separado.

---

## 5) Verificação de qualidade

## 5.1 Testes automáticos

Executado:

```bash
go test ./...
```

Resultado:
- `app`: OK
- `model`: OK
- `store`: OK (incluindo novos testes de recovery/backup rotativo)

Também executado especificamente:

```bash
go test ./store -run LoadWithRecovery -v
```

Resultado:
- `TestLoadWithRecoveryRestoresFromBackup`: PASS
- `TestLoadWithRecoveryWithoutBackupStartsEmpty`: PASS

## 5.2 Build

Executado:

```bash
go build ./...
```

Resultado: **sucesso**.

## 5.3 Smoke test manual (documentado)

Sessão manual em PTY com estado `smoke-final.json` cobrindo cenários principais:

1. criar lista (`a` em Listas)
2. criar tarefas (`a` em Tarefas)
3. reordenar (`K`)
4. ajustar prioridade (`4`)
5. marcar concluída (`x`)
6. alternar filtro (`f`)
7. busca (`/`, `Enter`, `Esc`)
8. excluir e desfazer (`d`, `y`, `u`)
9. renomear lista (`r`)

Evidência pós-smoke (`smoke-final.json`):
- lista renomeada para `Inbox Pessoal`
- tarefas persistidas com ordem manual (`position`) e prioridade alta (`priority: 3`) em uma delas
- status `done` persistido
- backups rotativos gerados e limitados a 10 arquivos

---

## 6) Arquivos alterados

- `cmd/todo/main.go`
- `tui/tui.go`
- `store/store.go`
- `store/store_test.go`
- `README.md`
- `POLISHING_REPORT.md` (novo)

---

## 7) Limitações conhecidas

1. Busca textual não é incremental caractere-a-caractere; aplica no `Enter` (mantendo simplicidade do fluxo).
2. `c` (troca de cor da lista) é atalho extra fora do escopo estrito do SPEC, mas mantido por utilidade e baixo custo.
3. `ARCHITECTURE.md` está desatualizado em relação ao código Go atual (não bloqueia uso, mas merece alinhamento documental em ciclo futuro).

---

## 8) Próximos passos opcionais (sem inflar escopo)

1. Ajustar busca para modo incremental (sem persistência em disco a cada tecla).
2. Alinhar `ARCHITECTURE.md` ao estado real do projeto.
3. Adicionar testes de integração para keymap da TUI (golden/state-driven).

---

## Estado final

**Pronto para uso diário local** no escopo MVP+ definido, com melhoria clara de robustez e consistência de UX sem overengineering.
