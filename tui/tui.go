package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"todo-cli/app"
	"todo-cli/model"
	"todo-cli/store"
)

type focusPane int

const (
	focusLists focusPane = iota
	focusTasks
)

func (f focusPane) String() string {
	if f == focusTasks {
		return "tarefas"
	}
	return "listas"
}

type uiMode int

const (
	modeNormal uiMode = iota
	modeAddList
	modeAddTask
	modeRenameList
	modeEditTask
	modeSearch
	modeConfirmDelete
	modeConfirmArchive
)

type deleteKind int

const (
	deleteNone deleteKind = iota
	deleteList
	deleteTask
	deleteAllTasks
)

type Model struct {
	svc       *app.Service
	statePath string

	focus         focusPane
	mode          uiMode
	listCursor    int
	taskCursor    int
	historyCursor int
	input         string

	confirmKind deleteKind
	confirmID   string
	confirmName string

	archiveListID   string
	archiveListName string
	archiveCount    int
	archiveAll      bool

	showHistory bool
	showHelp    bool

	status    string
	statusErr bool

	width  int
	height int

	palette []string
}

func NewModel(svc *app.Service, statePath, startupStatus string) *Model {
	status := strings.TrimSpace(startupStatus)
	if status == "" {
		status = "Pronto"
	}

	m := &Model{
		svc:       svc,
		statePath: statePath,
		focus:     focusLists,
		mode:      modeNormal,
		status:    status,
		palette:   []string{"blue", "green", "yellow", "magenta", "cyan", "red"},
	}
	m.restoreSessionContext()
	m.ensureSelection()

	if startupStatus == "" && m.shouldShowOnboarding() {
		m.setStatus("Bem-vindo. Pressione 'a' em Listas para criar sua primeira lista.", false)
	}

	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch m.mode {
		case modeAddList, modeAddTask, modeRenameList, modeEditTask, modeSearch:
			m.updateInputMode(msg)
		case modeConfirmDelete, modeConfirmArchive:
			m.updateConfirmMode(msg)
		default:
			if quit := m.updateNormalMode(msg); quit {
				_ = m.persistContextSilently()
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *Model) updateNormalMode(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "ctrl+c", "q":
		return true
	case "tab":
		if m.focus == focusLists {
			m.focus = focusTasks
		} else {
			m.focus = focusLists
		}
		_ = m.persistContextSilently()
		m.setStatus(fmt.Sprintf("Foco em %s", m.focus.String()), false)
	case "j", "down":
		m.moveCursor(1)
	case "k", "up":
		m.moveCursor(-1)
	case "enter":
		m.handleEnter()
	case "a":
		m.startAdd()
	case "r":
		m.startRenameList()
	case "e":
		m.startEditTask()
	case "x":
		m.toggleTaskDone()
	case "d":
		m.startDeleteConfirm()
	case "u":
		m.undo()
	case "f":
		m.cycleFilter()
	case "J":
		m.moveSelected(1)
	case "K":
		m.moveSelected(-1)
	case "1":
		m.setSelectedTaskPriority(model.PriorityNone)
	case "2":
		m.setSelectedTaskPriority(model.PriorityLow)
	case "3":
		m.setSelectedTaskPriority(model.PriorityMedium)
	case "4":
		m.setSelectedTaskPriority(model.PriorityHigh)
	case "c":
		m.cycleListColor()
	case "C":
		m.startArchiveConfirm()
	case "A":
		m.startArchiveAllConfirm()
	case "D":
		m.startDeleteAllConfirm()
	case "y":
		m.copyActiveTodos()
	case "h":
		m.toggleHistory()
	case "/":
		m.mode = modeSearch
		m.input = m.svc.State().Query
		m.setStatus("Busca incremental ativa: digite para filtrar em tempo real", false)
	case "?":
		m.showHelp = !m.showHelp
		if m.showHelp {
			m.setStatus("Atalhos abertos (pressione ? ou Esc para fechar)", false)
		} else {
			m.setStatus("Atalhos ocultos", false)
		}
	case "esc":
		if m.showHelp {
			m.showHelp = false
			m.setStatus("Atalhos ocultos", false)
			break
		}
		if strings.TrimSpace(m.svc.State().Query) != "" {
			m.svc.SetQuery("")
			m.taskCursor = 0
			m.persist("Busca limpa")
		}
	}

	m.ensureSelection()
	return false
}

func (m *Model) updateInputMode(msg tea.KeyMsg) {
	switch msg.String() {
	case "ctrl+c":
		if m.mode == modeSearch {
			m.svc.SetQuery("")
			m.taskCursor = 0
			m.persist("Busca limpa")
		}
		m.mode = modeNormal
		m.input = ""
		m.setStatus("Cancelado", false)
		return
	case "esc":
		if m.mode == modeSearch {
			m.svc.SetQuery("")
			m.taskCursor = 0
			m.persist("Busca limpa")
		} else {
			m.setStatus("Cancelado", false)
		}
		m.mode = modeNormal
		m.input = ""
		return
	case "enter":
		m.applyInput()
		return
	}

	switch msg.Type {
	case tea.KeyBackspace, tea.KeyCtrlH:
		m.input = trimLastRune(m.input)
	case tea.KeySpace:
		m.input += " "
	case tea.KeyRunes:
		m.input += string(msg.Runes)
	}

	if m.mode == modeSearch {
		m.svc.SetQuery(strings.TrimSpace(m.input))
		m.taskCursor = 0
		m.ensureSelection()
	}
}

func (m *Model) updateConfirmMode(msg tea.KeyMsg) {
	switch strings.ToLower(msg.String()) {
	case "y":
		if m.mode == modeConfirmArchive {
			m.confirmArchive()
			return
		}
		m.confirmDelete()
	case "n", "esc", "enter":
		if m.mode == modeConfirmArchive {
			m.archiveAll = false
			m.archiveListID = ""
			m.archiveListName = ""
			m.archiveCount = 0
		}
		if m.mode == modeConfirmDelete {
			m.confirmKind = deleteNone
			m.confirmID = ""
			m.confirmName = ""
		}
		m.mode = modeNormal
		m.setStatus("Ação cancelada", false)
	}
}

func (m *Model) applyInput() {
	text := strings.TrimSpace(m.input)
	switch m.mode {
	case modeAddList:
		if text == "" {
			m.setStatus("Nome da lista não pode ser vazio", true)
			return
		}
		color := m.palette[len(m.svc.Lists())%len(m.palette)]
		if _, err := m.svc.CreateList(text, color); err != nil {
			m.setStatus("Erro ao criar lista: "+err.Error(), true)
			return
		}
		m.listCursor = len(m.svc.Lists()) - 1
		m.mode = modeNormal
		m.input = ""
		m.persist("Lista criada")
	case modeAddTask:
		if text == "" {
			m.setStatus("Texto da tarefa não pode ser vazio", true)
			return
		}
		list, ok := m.activeList()
		if !ok {
			m.setStatus("Crie uma lista antes de adicionar tarefas", true)
			m.mode = modeNormal
			m.input = ""
			return
		}
		task, err := m.svc.CreateTask(list.ID, text)
		if err != nil {
			m.setStatus("Erro ao criar tarefa: "+err.Error(), true)
			return
		}
		m.mode = modeNormal
		m.input = ""
		m.taskCursor = m.indexOfTask(task.ID)
		m.persist("Tarefa criada")
	case modeRenameList:
		if text == "" {
			m.setStatus("Nome da lista não pode ser vazio", true)
			return
		}
		list, ok := m.activeList()
		if !ok {
			m.mode = modeNormal
			m.input = ""
			m.setStatus("Nenhuma lista selecionada", true)
			return
		}
		if _, err := m.svc.UpdateList(list.ID, text, list.Color); err != nil {
			m.setStatus("Erro ao renomear lista: "+err.Error(), true)
			return
		}
		m.mode = modeNormal
		m.input = ""
		m.persist("Lista renomeada")
	case modeEditTask:
		if text == "" {
			m.setStatus("Texto da tarefa não pode ser vazio", true)
			return
		}
		task, ok := m.selectedTask()
		if !ok {
			m.mode = modeNormal
			m.input = ""
			m.setStatus("Nenhuma tarefa selecionada", true)
			return
		}
		if _, err := m.svc.UpdateTask(task.ID, text); err != nil {
			m.setStatus("Erro ao editar tarefa: "+err.Error(), true)
			return
		}
		m.mode = modeNormal
		m.input = ""
		m.persist("Tarefa atualizada")
	case modeSearch:
		m.svc.SetQuery(text)
		m.mode = modeNormal
		m.input = ""
		m.taskCursor = 0
		if text == "" {
			m.persist("Busca limpa")
			return
		}
		m.persist("Busca aplicada")
	}
}

func (m *Model) moveCursor(delta int) {
	if m.focus == focusLists {
		lists := m.svc.Lists()
		if len(lists) == 0 {
			return
		}
		old := m.listCursor
		m.listCursor = clamp(m.listCursor+delta, 0, len(lists)-1)
		m.taskCursor = 0
		if m.listCursor != old {
			_ = m.persistContextSilently()
		}
		return
	}

	if m.showHistory {
		entries := m.archivedForDisplay()
		if len(entries) == 0 {
			return
		}
		m.historyCursor = clamp(m.historyCursor+delta, 0, len(entries)-1)
		return
	}

	tasks := m.visibleTasks()
	if len(tasks) == 0 {
		return
	}
	m.taskCursor = clamp(m.taskCursor+delta, 0, len(tasks)-1)
}

func (m *Model) handleEnter() {
	if m.focus != focusLists {
		return
	}
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Sem listas. Pressione 'a' para criar a primeira", false)
		return
	}
	m.taskCursor = 0
	_ = m.persistContextSilently()
	m.setStatus(fmt.Sprintf("Lista ativa: %s", list.Name), false)
}

func (m *Model) startAdd() {
	if m.focus == focusLists {
		m.mode = modeAddList
		m.input = ""
		return
	}

	if m.showHistory {
		m.setStatus("Feche o histórico ('h') para adicionar tarefas", false)
		return
	}

	if _, ok := m.activeList(); !ok {
		m.setStatus("Crie uma lista antes de adicionar tarefas", true)
		return
	}
	m.mode = modeAddTask
	m.input = ""
}

func (m *Model) startRenameList() {
	if m.focus != focusLists {
		m.setStatus("Renomear lista: mude o foco para Listas (Tab)", false)
		return
	}
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Nenhuma lista selecionada", true)
		return
	}
	m.mode = modeRenameList
	m.input = list.Name
}

func (m *Model) startEditTask() {
	if m.focus != focusTasks {
		m.setStatus("Editar tarefa: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Histórico é somente leitura. Pressione 'h' para voltar.", false)
		return
	}
	task, ok := m.selectedTask()
	if !ok {
		m.setStatus("Nenhuma tarefa selecionada", true)
		return
	}
	m.mode = modeEditTask
	m.input = task.Text
}

func (m *Model) toggleTaskDone() {
	if m.focus != focusTasks {
		m.setStatus("Marcar tarefa: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Histórico é somente leitura. Pressione 'h' para voltar.", false)
		return
	}
	task, ok := m.selectedTask()
	if !ok {
		m.setStatus("Nenhuma tarefa selecionada", true)
		return
	}
	updated, err := m.svc.ToggleDone(task.ID)
	if err != nil {
		m.setStatus("Erro ao alternar tarefa: "+err.Error(), true)
		return
	}
	if updated.Done {
		m.persist("Tarefa concluída")
	} else {
		m.persist("Tarefa reaberta")
	}
}

func (m *Model) moveSelected(delta int) {
	if m.focus == focusLists {
		m.moveSelectedList(delta)
		return
	}
	if m.showHistory {
		m.moveCursor(delta)
		return
	}
	m.moveSelectedTask(delta)
}

func (m *Model) moveSelectedList(delta int) {
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Nenhuma lista selecionada", true)
		return
	}

	var err error
	if delta < 0 {
		_, err = m.svc.MoveListUp(list.ID)
	} else {
		_, err = m.svc.MoveListDown(list.ID)
	}
	if err != nil {
		switch err {
		case app.ErrListAlreadyAtTop:
			m.setStatus("A lista já está no topo", false)
		case app.ErrListAlreadyAtBottom:
			m.setStatus("A lista já está no fim", false)
		default:
			m.setStatus("Erro ao mover lista: "+err.Error(), true)
		}
		return
	}

	if delta < 0 {
		m.listCursor--
	} else {
		m.listCursor++
	}
	m.ensureSelection()
	m.persist("Ordem das listas atualizada")
}

func (m *Model) moveSelectedTask(delta int) {
	if m.focus != focusTasks {
		m.setStatus("Ordenar tarefa: mude o foco para Tarefas (Tab)", false)
		return
	}
	task, ok := m.selectedTask()
	if !ok {
		m.setStatus("Nenhuma tarefa selecionada", true)
		return
	}

	var err error
	if delta < 0 {
		_, err = m.svc.MoveTaskUp(task.ID)
	} else {
		_, err = m.svc.MoveTaskDown(task.ID)
	}
	if err != nil {
		switch err {
		case app.ErrTaskAlreadyAtTop:
			m.setStatus("A tarefa já está no topo", false)
		case app.ErrTaskAlreadyAtBottom:
			m.setStatus("A tarefa já está no fim", false)
		default:
			m.setStatus("Erro ao mover tarefa: "+err.Error(), true)
		}
		return
	}

	if delta < 0 {
		m.taskCursor--
	} else {
		m.taskCursor++
	}
	m.ensureSelection()
	m.persist("Ordem das tarefas atualizada")
}

func (m *Model) setSelectedTaskPriority(priority model.Priority) {
	if m.focus != focusTasks {
		m.setStatus("Prioridade: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Histórico é somente leitura. Pressione 'h' para voltar.", false)
		return
	}
	task, ok := m.selectedTask()
	if !ok {
		m.setStatus("Nenhuma tarefa selecionada", true)
		return
	}
	updated, err := m.svc.SetTaskPriority(task.ID, priority)
	if err != nil {
		m.setStatus("Erro ao ajustar prioridade: "+err.Error(), true)
		return
	}
	m.persist(fmt.Sprintf("Prioridade: %s", priorityLabel(updated.Priority)))
}

func (m *Model) undo() {
	if err := m.svc.Undo(); err != nil {
		if err == app.ErrNothingToUndo {
			m.setStatus("Nada para desfazer", false)
			return
		}
		m.setStatus("Erro ao desfazer: "+err.Error(), true)
		return
	}
	m.showHistory = false
	m.persist("Undo aplicado")
}

func (m *Model) cycleFilter() {
	if m.focus != focusTasks {
		m.setStatus("Filtro de tarefas: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Filtro vale para tarefas ativas. Pressione 'h' para voltar.", false)
		return
	}
	if _, ok := m.activeList(); !ok {
		m.setStatus("Crie uma lista para usar filtros", false)
		return
	}
	st := m.svc.State()
	next := model.FilterAll
	switch st.Filter {
	case model.FilterAll:
		next = model.FilterTodo
	case model.FilterTodo:
		next = model.FilterDone
	case model.FilterDone:
		next = model.FilterAll
	}
	if err := m.svc.SetFilter(next); err != nil {
		m.setStatus("Erro ao alterar filtro: "+err.Error(), true)
		return
	}
	m.taskCursor = 0
	m.persist("Filtro: " + filterLabel(next))
}

func (m *Model) cycleListColor() {
	if m.focus != focusLists {
		m.setStatus("Cor da lista: mude o foco para Listas (Tab)", false)
		return
	}
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Nenhuma lista selecionada", true)
		return
	}
	current := 0
	for i, c := range m.palette {
		if c == strings.ToLower(strings.TrimSpace(list.Color)) {
			current = i
			break
		}
	}
	nextColor := m.palette[(current+1)%len(m.palette)]
	if _, err := m.svc.UpdateList(list.ID, list.Name, nextColor); err != nil {
		m.setStatus("Erro ao trocar cor da lista: "+err.Error(), true)
		return
	}
	m.persist("Cor da lista alterada")
}

func (m *Model) startArchiveConfirm() {
	if m.focus != focusTasks {
		m.setStatus("Arquivar concluídas: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Você já está no histórico. Pressione 'h' para voltar.", false)
		return
	}
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Nenhuma lista ativa", true)
		return
	}

	count := 0
	for _, t := range m.svc.Tasks(list.ID) {
		if t.Done {
			count++
		}
	}
	if count == 0 {
		m.setStatus("Não há tarefas concluídas para arquivar nesta lista", false)
		return
	}

	m.mode = modeConfirmArchive
	m.archiveListID = list.ID
	m.archiveListName = list.Name
	m.archiveCount = count
	m.archiveAll = false
}

func (m *Model) startArchiveAllConfirm() {
	if m.focus != focusTasks {
		m.setStatus("Arquivar todos: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Feche o histórico ('h') para arquivar todos os to-dos", false)
		return
	}
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Nenhuma lista ativa", true)
		return
	}
	count := len(m.svc.Tasks(list.ID))
	if count == 0 {
		m.setStatus("Não há to-dos para arquivar nesta lista", false)
		return
	}

	m.mode = modeConfirmArchive
	m.archiveListID = list.ID
	m.archiveListName = list.Name
	m.archiveCount = count
	m.archiveAll = true
}

func (m *Model) confirmArchive() {
	var (
		count int
		err   error
	)
	if m.archiveAll {
		count, err = m.svc.ArchiveAllToArchive(m.archiveListID)
	} else {
		count, err = m.svc.ClearCompletedToArchive(m.archiveListID)
	}
	if err != nil {
		m.mode = modeNormal
		m.archiveListID = ""
		m.archiveListName = ""
		m.archiveCount = 0
		m.archiveAll = false
		m.setStatus("Erro ao arquivar: "+err.Error(), true)
		return
	}
	m.mode = modeNormal
	m.archiveListID = ""
	m.archiveListName = ""
	m.archiveCount = 0
	m.archiveAll = false
	m.taskCursor = 0
	if m.showHistory {
		m.persist(fmt.Sprintf("%d itens arquivados • u desfaz", count))
		return
	}
	m.persist(fmt.Sprintf("%d to-dos arquivados • u desfaz", count))
}

func (m *Model) toggleHistory() {
	if m.focus != focusTasks {
		m.setStatus("Histórico: mude o foco para Tarefas (Tab)", false)
		return
	}
	if !m.showHistory && len(m.svc.ArchivedCompleted()) == 0 {
		m.setStatus("Histórico vazio. Arquive concluídas com 'C'.", false)
		return
	}
	m.showHistory = !m.showHistory
	m.historyCursor = 0
	if m.showHistory {
		m.setStatus("Histórico de concluídas aberto", false)
	} else {
		m.setStatus("Voltando para tarefas ativas", false)
	}
}

func (m *Model) startDeleteAllConfirm() {
	if m.focus != focusTasks {
		m.setStatus("Deletar todos: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Histórico é somente leitura. Pressione 'h' para voltar.", false)
		return
	}
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Nenhuma lista ativa", true)
		return
	}
	count := len(m.svc.Tasks(list.ID))
	if count == 0 {
		m.setStatus("Não há to-dos para deletar nesta lista", false)
		return
	}

	m.mode = modeConfirmDelete
	m.confirmKind = deleteAllTasks
	m.confirmID = list.ID
	m.confirmName = fmt.Sprintf("%s (%d to-dos)", list.Name, count)
}

func (m *Model) copyActiveTodos() {
	if m.focus != focusTasks {
		m.setStatus("Copiar to-dos: mude o foco para Tarefas (Tab)", false)
		return
	}
	if m.showHistory {
		m.setStatus("Histórico é somente leitura. Pressione 'h' para voltar.", false)
		return
	}
	list, ok := m.activeList()
	if !ok {
		m.setStatus("Nenhuma lista ativa", true)
		return
	}

	tasks := m.svc.Tasks(list.ID)
	if len(tasks) == 0 {
		m.setStatus("Sem to-dos ativos para copiar", false)
		return
	}

	parts := make([]string, 0, len(tasks))
	for _, t := range tasks {
		text := strings.TrimSpace(strings.ReplaceAll(t.Text, "\n", " "))
		if text == "" {
			continue
		}
		parts = append(parts, "- "+text)
	}
	if len(parts) == 0 {
		m.setStatus("Sem to-dos válidos para copiar", false)
		return
	}

	payload := strings.Join(parts, "\n")
	if err := copyToClipboard(payload); err != nil {
		m.setStatus("Falha ao copiar: "+err.Error(), true)
		return
	}
	m.setStatus(fmt.Sprintf("%d to-dos copiados para a área de transferência", len(parts)), false)
}

func (m *Model) startDeleteConfirm() {
	if m.focus == focusLists {
		list, ok := m.activeList()
		if !ok {
			m.setStatus("Nenhuma lista selecionada", true)
			return
		}
		count := len(m.svc.Tasks(list.ID))
		name := list.Name
		if count > 0 {
			name = fmt.Sprintf("%s (%d tarefas)", list.Name, count)
		}
		m.mode = modeConfirmDelete
		m.confirmKind = deleteList
		m.confirmID = list.ID
		m.confirmName = name
		return
	}

	if m.showHistory {
		m.setStatus("Histórico é somente leitura. Pressione 'h' para voltar.", false)
		return
	}

	task, ok := m.selectedTask()
	if !ok {
		m.setStatus("Nenhuma tarefa selecionada", true)
		return
	}
	m.mode = modeConfirmDelete
	m.confirmKind = deleteTask
	m.confirmID = task.ID
	m.confirmName = task.Text
}

func (m *Model) confirmDelete() {
	switch m.confirmKind {
	case deleteList:
		if err := m.svc.DeleteList(m.confirmID); err != nil {
			m.setStatus("Erro ao excluir lista: "+err.Error(), true)
			break
		}
		m.persist("Lista excluída • u desfaz")
	case deleteTask:
		if err := m.svc.DeleteTask(m.confirmID); err != nil {
			m.setStatus("Erro ao excluir tarefa: "+err.Error(), true)
			break
		}
		m.persist("Tarefa excluída • u desfaz")
	case deleteAllTasks:
		count, err := m.svc.DeleteAllTasks(m.confirmID)
		if err != nil {
			m.setStatus("Erro ao deletar to-dos: "+err.Error(), true)
			break
		}
		m.taskCursor = 0
		m.persist(fmt.Sprintf("%d to-dos deletados • u desfaz", count))
	}
	m.mode = modeNormal
	m.confirmKind = deleteNone
	m.confirmID = ""
	m.confirmName = ""
	m.ensureSelection()
}

func (m *Model) persist(success string) {
	m.svc.MarkOnboardingSeen()
	if err := m.svc.SetSessionContext(m.currentActiveListID(), m.sessionFocusValue()); err != nil {
		m.setStatus("Falha ao atualizar contexto da sessão: "+err.Error(), true)
		return
	}
	if err := store.Autosave(m.statePath, m.svc.State()); err != nil {
		m.setStatus("Alteração aplicada, mas falhou ao salvar em disco: "+err.Error(), true)
		return
	}
	m.ensureSelection()
	m.setStatus(success, false)
}

func (m *Model) persistContextSilently() error {
	if err := m.svc.SetSessionContext(m.currentActiveListID(), m.sessionFocusValue()); err != nil {
		m.setStatus("Falha ao atualizar contexto da sessão: "+err.Error(), true)
		return err
	}
	if err := store.Autosave(m.statePath, m.svc.State()); err != nil {
		m.setStatus("Falha ao salvar contexto da sessão: "+err.Error(), true)
		return err
	}
	return nil
}

func (m *Model) setStatus(text string, isErr bool) {
	m.status = text
	m.statusErr = isErr
}

func (m *Model) restoreSessionContext() {
	st := m.svc.State()
	if st.Metadata.Session.Focus == model.SessionFocusTasks {
		m.focus = focusTasks
	}

	if st.Metadata.Session.ActiveListID != "" {
		lists := m.svc.Lists()
		for i, l := range lists {
			if l.ID == st.Metadata.Session.ActiveListID {
				m.listCursor = i
				break
			}
		}
	}
}

func (m *Model) shouldShowOnboarding() bool {
	st := m.svc.State()
	if st.Metadata.FirstRun {
		return true
	}
	return len(st.Lists) == 0 && len(st.Tasks) == 0
}

func (m *Model) ensureSelection() {
	lists := m.svc.Lists()
	if len(lists) == 0 {
		m.listCursor = 0
		m.taskCursor = 0
		m.historyCursor = 0
		m.focus = focusLists
		return
	}
	m.listCursor = clamp(m.listCursor, 0, len(lists)-1)

	if m.showHistory {
		entries := m.archivedForDisplay()
		if len(entries) == 0 {
			m.historyCursor = 0
			return
		}
		m.historyCursor = clamp(m.historyCursor, 0, len(entries)-1)
		return
	}

	tasks := m.visibleTasks()
	if len(tasks) == 0 {
		m.taskCursor = 0
		return
	}
	m.taskCursor = clamp(m.taskCursor, 0, len(tasks)-1)
}

func (m *Model) activeList() (model.List, bool) {
	lists := m.svc.Lists()
	if len(lists) == 0 {
		return model.List{}, false
	}
	if m.listCursor < 0 || m.listCursor >= len(lists) {
		m.listCursor = 0
	}
	return lists[m.listCursor], true
}

func (m *Model) currentActiveListID() string {
	list, ok := m.activeList()
	if !ok {
		return ""
	}
	return list.ID
}

func (m *Model) sessionFocusValue() string {
	if m.focus == focusTasks {
		return model.SessionFocusTasks
	}
	return model.SessionFocusLists
}

func (m *Model) visibleTasks() []model.Task {
	list, ok := m.activeList()
	if !ok {
		return []model.Task{}
	}
	all := m.svc.Tasks(list.ID)
	state := m.svc.State()
	query := strings.ToLower(strings.TrimSpace(state.Query))

	out := make([]model.Task, 0, len(all))
	for _, t := range all {
		if !matchesFilter(state.Filter, t.Done) {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(t.Text), query) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (m *Model) selectedTask() (model.Task, bool) {
	tasks := m.visibleTasks()
	if len(tasks) == 0 {
		return model.Task{}, false
	}
	if m.taskCursor < 0 || m.taskCursor >= len(tasks) {
		m.taskCursor = 0
	}
	return tasks[m.taskCursor], true
}

func (m *Model) archivedForDisplay() []model.ArchivedCompletedTask {
	all := m.svc.ArchivedCompleted()
	if len(all) == 0 {
		return all
	}

	list, hasList := m.activeList()
	out := make([]model.ArchivedCompletedTask, 0, len(all))
	for i := len(all) - 1; i >= 0; i-- {
		e := all[i]
		if hasList {
			if e.OriginListID != "" {
				if e.OriginListID != list.ID {
					continue
				}
			} else if strings.TrimSpace(e.OriginList) != strings.TrimSpace(list.Name) {
				// fallback para entradas legadas sem OriginListID
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

func (m *Model) indexOfTask(taskID string) int {
	tasks := m.visibleTasks()
	for i, t := range tasks {
		if t.ID == taskID {
			return i
		}
	}
	if len(tasks) == 0 {
		return 0
	}
	return len(tasks) - 1
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "carregando..."
	}

	st := m.svc.State()
	title := lipgloss.NewStyle().Bold(true).Render("todo-cli")
	summary := fmt.Sprintf("foco: %s • filtro: %s", m.focus.String(), filterLabel(st.Filter))
	if st.Query != "" {
		summary += " • busca: \"" + st.Query + "\""
	}
	if m.showHistory {
		summary += " • histórico: ligado"
	}
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		title,
		lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("  "+summary),
	)

	viewW := m.viewportWidth()
	const paneGap = 1
	const rightInset = 6
	outerPaneW := viewW - rightInset
	if outerPaneW < 40 {
		outerPaneW = viewW
	}
	innerPaneW := outerPaneW - 2
	if innerPaneW < 20 {
		innerPaneW = outerPaneW
	}

	panelH := m.height - 6
	if panelH < 8 {
		panelH = 8
	}
	innerPaneH := panelH - 2
	if innerPaneH < 6 {
		innerPaneH = 6
	}

	leftW, rightW := m.paneWidths(innerPaneW, paneGap)
	split := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderListsPanel(leftW, innerPaneH),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│"),
		m.renderTasksPanel(rightW, innerPaneH),
	)

	frameColor := lipgloss.Color("240")
	if m.mode == modeNormal {
		frameColor = lipgloss.Color("39")
	}
	panes := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(frameColor).
		Width(outerPaneW).
		Height(panelH).
		Render(split)

	if outerPaneW < viewW {
		panes = lipgloss.JoinHorizontal(lipgloss.Top, panes, strings.Repeat(" ", viewW-outerPaneW))
	}

	statusText := m.status
	if statusText == "" {
		statusText = "Pronto"
	}
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	if m.statusErr {
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	}

	rightHint := "? atalhos"
	if m.showHelp {
		rightHint = "Esc/? fechar atalhos"
	}
	footerLine := m.renderFooter(statusText, statusStyle, rightHint)

	promptLine := ""
	switch m.mode {
	case modeAddList:
		promptLine = "Nova lista: " + m.input + "▌"
	case modeAddTask:
		promptLine = "Nova tarefa: " + m.input + "▌"
	case modeRenameList:
		promptLine = "Renomear lista: " + m.input + "▌"
	case modeEditTask:
		promptLine = "Editar tarefa: " + m.input + "▌"
	case modeSearch:
		promptLine = "Busca (/): " + m.input + "▌  (incremental; Enter confirma, Esc limpa)"
	case modeConfirmDelete:
		target := "item"
		if m.confirmKind == deleteList {
			target = "lista"
		} else if m.confirmKind == deleteTask {
			target = "tarefa"
		} else if m.confirmKind == deleteAllTasks {
			target = "todos os to-dos"
		}
		promptLine = fmt.Sprintf("Excluir %s \"%s\"? [y/N]", target, m.confirmName)
	case modeConfirmArchive:
		if m.archiveAll {
			promptLine = fmt.Sprintf("Arquivar TODOS os %d to-dos da lista \"%s\"? [y/N]", m.archiveCount, m.archiveListName)
		} else {
			promptLine = fmt.Sprintf("Arquivar %d concluídas da lista \"%s\"? [y/N]", m.archiveCount, m.archiveListName)
		}
	}
	if promptLine != "" {
		promptLine = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Width(viewW).Render(promptLine)
	}

	parts := []string{header}
	if m.shouldShowOnboarding() {
		parts = append(parts, m.renderOnboarding(viewW))
	}

	if m.showHelp {
		popupW := viewW - 8
		if popupW > 96 {
			popupW = 96
		}
		if popupW < 56 {
			popupW = viewW - 2
		}
		if popupW < 40 {
			popupW = 40
		}
		popup := m.renderHelpOverlay(popupW)
		panes = lipgloss.Place(viewW, panelH, lipgloss.Center, lipgloss.Center, popup)
	}

	parts = append(parts, panes, footerLine)
	if promptLine != "" && !m.showHelp {
		parts = append(parts, promptLine)
	}
	return strings.Join(parts, "\n")
}

func (m *Model) viewportWidth() int {
	if m.width <= 0 {
		return 1
	}
	// Reservamos 1 coluna para evitar clipping/wrap no último caractere
	// em alguns terminais/TUIs (borda direita “sumindo”).
	if m.width > 1 {
		return m.width - 1
	}
	return m.width
}

func (m *Model) paneWidths(total, gap int) (int, int) {
	if total <= 0 {
		return 24, 30
	}
	if gap < 0 {
		gap = 0
	}

	minLeft := 20
	minRight := 30
	if total < minLeft+minRight+gap {
		left := total / 3
		if left < 12 {
			left = 12
		}
		right := total - left - gap
		if right < 12 {
			right = 12
			left = total - right - gap
			if left < 10 {
				left = 10
			}
		}
		return left, right
	}

	left := total / 4
	if left < 22 {
		left = 22
	}
	if left > 34 {
		left = 34
	}

	right := total - left - gap
	if right < minRight {
		right = minRight
		left = total - right - gap
	}
	if left < minLeft {
		left = minLeft
		right = total - left - gap
	}

	return left, right
}

func (m *Model) renderFooter(statusText string, statusStyle lipgloss.Style, rightHint string) string {
	left := strings.TrimSpace(statusText)
	right := strings.TrimSpace(rightHint)
	if left == "" {
		left = "Pronto"
	}
	if right == "" {
		right = "? atalhos"
	}

	leftW := utf8.RuneCountInString(left)
	rightW := utf8.RuneCountInString(right)
	width := m.viewportWidth()
	if width <= 0 {
		width = leftW + rightW + 2
	}

	if leftW+rightW+1 > width {
		maxLeft := width - rightW - 1
		if maxLeft < 8 {
			maxLeft = 8
		}
		left = truncateRunes(left, maxLeft)
		leftW = utf8.RuneCountInString(left)
	}

	padding := width - leftW - rightW
	if padding < 1 {
		padding = 1
	}

	rightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	line := statusStyle.Render(left) + strings.Repeat(" ", padding) + rightStyle.Render(right)
	return lipgloss.NewStyle().Width(width).Render(line)
}

func (m *Model) renderHelpOverlay(width int) string {
	title := lipgloss.NewStyle().Bold(true).Render("Atalhos")
	section := lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)
	line := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	rows := []string{
		title,
		"",
		section.Render("Globais"),
		line.Render("  Tab alterna foco • j/k navega • q sai"),
		line.Render("  / busca • u desfaz • ? abre/fecha atalhos • Esc fecha"),
		"",
		section.Render("Listas (com foco em Listas)"),
		line.Render("  a cria • r renomeia • c cor • J/K reordena • d exclui"),
		line.Render("  Enter define lista ativa"),
		"",
		section.Render("Tarefas (com foco em Tarefas)"),
		line.Render("  a cria • e edita • x conclui/reabre • 1..4 prioridade"),
		line.Render("  J/K reordena • f filtro • y copia to-dos ativos"),
		line.Render("  C arquiva concluídas • A arquiva todos • D deleta todos"),
		line.Render("  h histórico"),
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("244")).
		Padding(1, 2)

	return style.Width(width).Render(strings.Join(rows, "\n"))
}

func (m *Model) renderOnboarding(width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1)
	if width <= 0 {
		width = m.viewportWidth()
	}
	text := "Primeiro uso:\n1) Em Listas: 'a' para criar lista\n2) Tab para Tarefas e 'a' para adicionar\n3) 'x' conclui, 'C' arquiva concluídas, 'h' abre histórico"
	return style.Width(width).Render(text)
}

func (m *Model) contextualHelp() string {
	switch m.mode {
	case modeAddList, modeAddTask, modeRenameList, modeEditTask:
		return "Digite texto • Enter confirmar • Esc cancelar"
	case modeSearch:
		return "Busca incremental • Digite para filtrar • Enter confirma • Esc limpa"
	case modeConfirmDelete, modeConfirmArchive:
		return "Confirmar ação • y confirma • n/Esc cancela"
	}

	if m.showHistory {
		return "Histórico • j/k navegar • h voltar • Tab foco • u undo • q sair"
	}

	if m.focus == focusLists {
		return "Listas • a criar • r renomear • c cor • J/K reordenar • d excluir • Enter ativar • Tab tarefas • q sair"
	}
	return "Tarefas • a criar • e editar • x done • 1..4 prioridade • J/K reordenar • f filtro • / busca • C arquivar concluídas • h histórico • u undo"
}

func (m *Model) renderListsPanel(width, height int) string {
	lists := m.svc.Lists()
	panelTitle := "Listas"

	lines := make([]string, 0, len(lists)+2)
	lines = append(lines, panelTitleStyled(panelTitle, m.focus == focusLists))
	if len(lists) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Sem listas. Pressione 'a' para criar a primeira."))
	} else {
		for i, l := range lists {
			cursor := " "
			if i == m.listCursor {
				cursor = "▸"
			}
			dot := lipgloss.NewStyle().Foreground(colorForName(l.Color)).Render("●")
			line := fmt.Sprintf("%s %s %s", cursor, dot, l.Name)
			if i == m.listCursor {
				style := lipgloss.NewStyle().Bold(true)
				if m.focus == focusLists {
					style = style.Foreground(lipgloss.Color("229"))
				}
				line = style.Render(line)
			}
			lines = append(lines, line)
		}
	}

	panelStyle := lipgloss.NewStyle().
		Width(width).
		Height(height)
	return panelStyle.Render(strings.Join(lines, "\n"))
}

func (m *Model) renderTasksPanel(width, height int) string {
	if m.showHistory {
		return m.renderHistoryPanel(width, height)
	}

	list, hasList := m.activeList()
	allTasksInList := []model.Task{}
	if hasList {
		allTasksInList = m.svc.Tasks(list.ID)
	}
	tasks := m.visibleTasks()

	title := "Tarefas"
	if hasList {
		title = fmt.Sprintf("Tarefas — %s", list.Name)
	}

	lines := make([]string, 0, len(tasks)+3)
	lines = append(lines, panelTitleStyled(title, m.focus == focusTasks))

	if !hasList {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Sem lista ativa. Vá em Listas e pressione 'a'."))
	} else if len(tasks) == 0 {
		state := m.svc.State()
		switch {
		case len(allTasksInList) == 0:
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Lista vazia. Pressione 'a' para adicionar tarefa."))
		case strings.TrimSpace(state.Query) != "":
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Nenhuma tarefa corresponde à busca/filtro atual."))
		default:
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("Nenhuma tarefa para o filtro atual (use 'f')."))
		}
	} else {
		for i, t := range tasks {
			cursor := " "
			if i == m.taskCursor {
				cursor = "▸"
			}
			check := "[ ]"
			if t.Done {
				check = "[x]"
			}
			pri := priorityIndicator(t.Priority)

			cursorStyle := lipgloss.NewStyle()
			checkStyle := lipgloss.NewStyle()
			textStyle := lipgloss.NewStyle()

			// Evita glitch visual com ANSI em alguns terminais ao combinar
			// Strikethrough + segmentos já coloridos (indicador de prioridade).
			if t.Done {
				textStyle = textStyle.Faint(true)
			}
			if i == m.taskCursor {
				cursorStyle = cursorStyle.Bold(true)
				checkStyle = checkStyle.Bold(true)
				textStyle = textStyle.Bold(true)
				if m.focus == focusTasks {
					sel := lipgloss.Color("229")
					cursorStyle = cursorStyle.Foreground(sel)
					checkStyle = checkStyle.Foreground(sel)
					textStyle = textStyle.Foreground(sel)
				}
			}

			line := lipgloss.JoinHorizontal(lipgloss.Left,
				cursorStyle.Render(cursor+" "),
				checkStyle.Render(check+" "),
				pri+" ",
				textStyle.Render(t.Text),
			)
			lines = append(lines, line)
		}
	}

	panelStyle := lipgloss.NewStyle().
		Width(width).
		Height(height)
	return panelStyle.Render(strings.Join(lines, "\n"))
}

func (m *Model) renderHistoryPanel(width, height int) string {
	entries := m.archivedForDisplay()
	list, hasList := m.activeList()
	title := "Histórico de concluídas"
	if hasList {
		title = fmt.Sprintf("Histórico de concluídas — %s", list.Name)
	}

	lines := make([]string, 0, len(entries)+2)
	lines = append(lines, panelTitleStyled(title, m.focus == focusTasks))
	if len(entries) == 0 {
		emptyMsg := "Histórico vazio. Use 'C' para arquivar concluídas da lista ativa."
		if hasList {
			emptyMsg = "Sem itens arquivados para esta lista. Use 'C' ou 'A' na lista ativa."
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(emptyMsg))
	} else {
		for i, e := range entries {
			cursor := " "
			if i == m.historyCursor {
				cursor = "▸"
			}
			line := fmt.Sprintf("%s %s %s (%s • %s)", cursor, priorityIndicator(e.Priority), e.TaskText, e.OriginList, e.DoneAt.Local().Format("02/01 15:04"))
			style := lipgloss.NewStyle().Faint(true)
			if i == m.historyCursor {
				style = lipgloss.NewStyle().Bold(true)
				if m.focus == focusTasks {
					style = style.Foreground(lipgloss.Color("229"))
				}
			}
			lines = append(lines, style.Render(line))
		}
	}

	panelStyle := lipgloss.NewStyle().
		Width(width).
		Height(height)
	return panelStyle.Render(strings.Join(lines, "\n"))
}

func panelTitleStyled(title string, active bool) string {
	base := lipgloss.NewStyle().Bold(true)
	if !active {
		return base.Render(title)
	}
	text := base.Foreground(lipgloss.Color("229")).Render(title)
	marker := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("*")
	return lipgloss.JoinHorizontal(lipgloss.Left, text, " ", marker)
}

func matchesFilter(filter model.Filter, done bool) bool {
	switch filter {
	case model.FilterTodo:
		return !done
	case model.FilterDone:
		return done
	default:
		return true
	}
}

func priorityIndicator(p model.Priority) string {
	s := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("•")
	switch p {
	case model.PriorityLow:
		s = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("●")
	case model.PriorityMedium:
		s = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("●")
	case model.PriorityHigh:
		s = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("●")
	}
	return s
}

func priorityLabel(p model.Priority) string {
	switch p {
	case model.PriorityLow:
		return "baixa"
	case model.PriorityMedium:
		return "média"
	case model.PriorityHigh:
		return "alta"
	default:
		return "nenhuma"
	}
}

func filterLabel(f model.Filter) string {
	switch f {
	case model.FilterTodo:
		return "abertas"
	case model.FilterDone:
		return "concluídas"
	default:
		return "todas"
	}
}

func colorForName(name string) lipgloss.Color {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "blue":
		return lipgloss.Color("12")
	case "green":
		return lipgloss.Color("10")
	case "yellow":
		return lipgloss.Color("11")
	case "magenta", "purple":
		return lipgloss.Color("13")
	case "cyan":
		return lipgloss.Color("14")
	case "red":
		return lipgloss.Color("9")
	default:
		return lipgloss.Color("7")
	}
}

func copyToClipboard(text string) error {
	candidates := []struct {
		name string
		args []string
	}{
		{name: "wl-copy", args: []string{"--type", "text/plain"}},
		{name: "xclip", args: []string{"-in", "-selection", "clipboard"}},
		{name: "xsel", args: []string{"--clipboard", "--input"}},
		{name: "pbcopy"},
	}

	for _, c := range candidates {
		if _, err := exec.LookPath(c.name); err != nil {
			continue
		}
		// Execução assíncrona para evitar qualquer travamento da UI.
		go runClipboardCommand(c.name, c.args, text)
		return nil
	}
	return fmt.Errorf("nenhum comando de clipboard disponível (instale wl-copy ou xclip)")
}

func runClipboardCommand(name string, args []string, text string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(text)
	_ = cmd.Run()
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func trimLastRune(s string) string {
	if s == "" {
		return s
	}
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}
