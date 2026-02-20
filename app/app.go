package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"todo-cli/model"
)

const undoStackLimit = 20

var (
	ErrListNotFound        = errors.New("list not found")
	ErrTaskNotFound        = errors.New("task not found")
	ErrInvalidName         = errors.New("name must not be empty")
	ErrInvalidTask         = errors.New("task text must not be empty")
	ErrInvalidFilter       = errors.New("invalid filter")
	ErrInvalidPriority     = errors.New("invalid priority")
	ErrNothingToUndo       = errors.New("nothing to undo")
	ErrInvalidListRef      = errors.New("list id must not be empty")
	ErrTaskAlreadyAtTop    = errors.New("task is already at top")
	ErrTaskAlreadyAtBottom = errors.New("task is already at bottom")
	ErrListAlreadyAtTop    = errors.New("list is already at top")
	ErrListAlreadyAtBottom = errors.New("list is already at bottom")
	ErrNoCompletedToClear  = errors.New("no completed tasks to clear")
	ErrNoTasksInList       = errors.New("no tasks in list")
	ErrInvalidSessionFocus = errors.New("invalid session focus")
)

// Service holds domain rules and in-memory state.
type Service struct {
	state model.AppState
	undo  []model.AppState
}

// NewService creates a service with a copy of the provided state.
func NewService(state model.AppState) *Service {
	state = normalizeState(state)
	return &Service{state: state, undo: []model.AppState{}}
}

// State returns a copy of current state.
func (s *Service) State() model.AppState {
	return copyState(s.state)
}

// Lists returns all lists as a copy.
func (s *Service) Lists() []model.List {
	lists := make([]model.List, len(s.state.Lists))
	copy(lists, s.state.Lists)
	return lists
}

// GetList returns a list by id.
func (s *Service) GetList(id string) (model.List, error) {
	for _, l := range s.state.Lists {
		if l.ID == id {
			return l, nil
		}
	}
	return model.List{}, ErrListNotFound
}

// Tasks returns tasks for a list sorted by manual position.
// If listID is empty, returns all tasks sorted by list then position.
func (s *Service) Tasks(listID string) []model.Task {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		out := make([]model.Task, len(s.state.Tasks))
		copy(out, s.state.Tasks)
		sortTasks(out, "")
		return out
	}

	out := make([]model.Task, 0)
	for _, t := range s.state.Tasks {
		if t.ListID == listID {
			out = append(out, t)
		}
	}
	sortTasks(out, listID)
	return out
}

// GetTask returns a task by id.
func (s *Service) GetTask(id string) (model.Task, error) {
	for _, t := range s.state.Tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return model.Task{}, ErrTaskNotFound
}

func (s *Service) CreateList(name, color string) (model.List, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.List{}, ErrInvalidName
	}
	now := time.Now().UTC()
	list := model.List{
		ID:        newID(),
		Name:      name,
		Color:     strings.TrimSpace(color),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.pushUndo()
	s.state.Lists = append(s.state.Lists, list)
	return list, nil
}

func (s *Service) UpdateList(id, name, color string) (model.List, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.List{}, ErrInvalidName
	}
	for i := range s.state.Lists {
		if s.state.Lists[i].ID == id {
			s.pushUndo()
			s.state.Lists[i].Name = name
			s.state.Lists[i].Color = strings.TrimSpace(color)
			s.state.Lists[i].UpdatedAt = time.Now().UTC()
			return s.state.Lists[i], nil
		}
	}
	return model.List{}, ErrListNotFound
}

func (s *Service) DeleteList(id string) error {
	for i := range s.state.Lists {
		if s.state.Lists[i].ID != id {
			continue
		}

		s.pushUndo()
		s.state.Lists = append(s.state.Lists[:i], s.state.Lists[i+1:]...)

		keptTasks := make([]model.Task, 0, len(s.state.Tasks))
		for _, t := range s.state.Tasks {
			if t.ListID != id {
				keptTasks = append(keptTasks, t)
			}
		}
		s.state.Tasks = keptTasks
		return nil
	}
	return ErrListNotFound
}

func (s *Service) MoveListUp(listID string) (model.List, error) {
	return s.moveList(listID, -1)
}

func (s *Service) MoveListDown(listID string) (model.List, error) {
	return s.moveList(listID, 1)
}

func (s *Service) moveList(listID string, direction int) (model.List, error) {
	idx := -1
	for i := range s.state.Lists {
		if s.state.Lists[i].ID == listID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return model.List{}, ErrListNotFound
	}

	target := idx + direction
	if target < 0 {
		return model.List{}, ErrListAlreadyAtTop
	}
	if target >= len(s.state.Lists) {
		return model.List{}, ErrListAlreadyAtBottom
	}

	s.pushUndo()
	s.state.Lists[idx], s.state.Lists[target] = s.state.Lists[target], s.state.Lists[idx]
	now := time.Now().UTC()
	s.state.Lists[idx].UpdatedAt = now
	s.state.Lists[target].UpdatedAt = now
	return s.state.Lists[target], nil
}

func (s *Service) CreateTask(listID, text string) (model.Task, error) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return model.Task{}, ErrInvalidListRef
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return model.Task{}, ErrInvalidTask
	}
	if !s.hasList(listID) {
		return model.Task{}, ErrListNotFound
	}
	now := time.Now().UTC()
	insertPos := s.nextTodoInsertPosition(listID)
	task := model.Task{
		ID:        newID(),
		ListID:    listID,
		Text:      text,
		Done:      false,
		Priority:  model.PriorityNone,
		Position:  insertPos,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.pushUndo()
	for i := range s.state.Tasks {
		if s.state.Tasks[i].ListID == listID && s.state.Tasks[i].Position >= insertPos {
			s.state.Tasks[i].Position++
		}
	}
	s.state.Tasks = append(s.state.Tasks, task)
	s.normalizePositionsForList(listID)
	return task, nil
}

func (s *Service) UpdateTask(id, text string) (model.Task, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return model.Task{}, ErrInvalidTask
	}
	for i := range s.state.Tasks {
		if s.state.Tasks[i].ID == id {
			s.pushUndo()
			s.state.Tasks[i].Text = text
			s.state.Tasks[i].UpdatedAt = time.Now().UTC()
			return s.state.Tasks[i], nil
		}
	}
	return model.Task{}, ErrTaskNotFound
}

func (s *Service) DeleteTask(id string) error {
	for i := range s.state.Tasks {
		if s.state.Tasks[i].ID != id {
			continue
		}
		listID := s.state.Tasks[i].ListID
		s.pushUndo()
		s.state.Tasks = append(s.state.Tasks[:i], s.state.Tasks[i+1:]...)
		s.normalizePositionsForList(listID)
		return nil
	}
	return ErrTaskNotFound
}

func (s *Service) ToggleDone(taskID string) (model.Task, error) {
	for i := range s.state.Tasks {
		if s.state.Tasks[i].ID == taskID {
			s.pushUndo()
			s.state.Tasks[i].Done = !s.state.Tasks[i].Done
			s.state.Tasks[i].UpdatedAt = time.Now().UTC()
			if s.state.Tasks[i].Done {
				listID := s.state.Tasks[i].ListID
				maxPos := 0
				for j := range s.state.Tasks {
					if j == i || s.state.Tasks[j].ListID != listID {
						continue
					}
					if s.state.Tasks[j].Position > maxPos {
						maxPos = s.state.Tasks[j].Position
					}
				}
				s.state.Tasks[i].Position = maxPos + 1
				s.normalizePositionsForList(listID)
			}
			return s.state.Tasks[i], nil
		}
	}
	return model.Task{}, ErrTaskNotFound
}

func (s *Service) SetTaskPriority(taskID string, priority model.Priority) (model.Task, error) {
	if priority < model.PriorityNone || priority > model.PriorityHigh {
		return model.Task{}, fmt.Errorf("%w: %d", ErrInvalidPriority, priority)
	}

	for i := range s.state.Tasks {
		if s.state.Tasks[i].ID == taskID {
			if s.state.Tasks[i].Priority == priority {
				return s.state.Tasks[i], nil
			}
			s.pushUndo()
			s.state.Tasks[i].Priority = priority
			s.state.Tasks[i].UpdatedAt = time.Now().UTC()
			return s.state.Tasks[i], nil
		}
	}
	return model.Task{}, ErrTaskNotFound
}

func (s *Service) MoveTaskUp(taskID string) (model.Task, error) {
	return s.moveTask(taskID, -1)
}

func (s *Service) MoveTaskDown(taskID string) (model.Task, error) {
	return s.moveTask(taskID, 1)
}

func (s *Service) moveTask(taskID string, direction int) (model.Task, error) {
	idx := -1
	for i := range s.state.Tasks {
		if s.state.Tasks[i].ID == taskID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return model.Task{}, ErrTaskNotFound
	}

	listID := s.state.Tasks[idx].ListID
	ordered := s.taskIndexesForList(listID)
	position := -1
	for i, taskIdx := range ordered {
		if s.state.Tasks[taskIdx].ID == taskID {
			position = i
			break
		}
	}
	if position == -1 {
		return model.Task{}, ErrTaskNotFound
	}

	targetPos := position + direction
	if targetPos < 0 {
		return model.Task{}, ErrTaskAlreadyAtTop
	}
	if targetPos >= len(ordered) {
		return model.Task{}, ErrTaskAlreadyAtBottom
	}

	s.pushUndo()
	a := ordered[position]
	b := ordered[targetPos]
	s.state.Tasks[a].Position, s.state.Tasks[b].Position = s.state.Tasks[b].Position, s.state.Tasks[a].Position
	now := time.Now().UTC()
	s.state.Tasks[a].UpdatedAt = now
	s.state.Tasks[b].UpdatedAt = now
	s.normalizePositionsForList(listID)

	for i := range s.state.Tasks {
		if s.state.Tasks[i].ID == taskID {
			return s.state.Tasks[i], nil
		}
	}
	return model.Task{}, ErrTaskNotFound
}

func (s *Service) ClearCompletedToArchive(listID string) (int, error) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return 0, ErrInvalidListRef
	}

	list, err := s.GetList(listID)
	if err != nil {
		return 0, err
	}

	toArchive := make([]model.ArchivedCompletedTask, 0)
	kept := make([]model.Task, 0, len(s.state.Tasks))
	now := time.Now().UTC()

	for _, t := range s.state.Tasks {
		if t.ListID == listID && t.Done {
			doneAt := t.UpdatedAt
			if doneAt.IsZero() {
				doneAt = now
			}
			toArchive = append(toArchive, model.ArchivedCompletedTask{
				ID:           newID(),
				TaskText:     t.Text,
				OriginListID: list.ID,
				OriginList:   list.Name,
				Priority:     t.Priority,
				DoneAt:       doneAt,
				ArchivedAt:   now,
			})
			continue
		}
		kept = append(kept, t)
	}

	if len(toArchive) == 0 {
		return 0, ErrNoCompletedToClear
	}

	s.pushUndo()
	s.state.Tasks = kept
	s.normalizePositionsForList(listID)
	s.state.ArchivedCompleted = append(s.state.ArchivedCompleted, toArchive...)
	return len(toArchive), nil
}

func (s *Service) ArchiveAllToArchive(listID string) (int, error) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return 0, ErrInvalidListRef
	}

	list, err := s.GetList(listID)
	if err != nil {
		return 0, err
	}

	toArchive := make([]model.ArchivedCompletedTask, 0)
	kept := make([]model.Task, 0, len(s.state.Tasks))
	now := time.Now().UTC()

	for _, t := range s.state.Tasks {
		if t.ListID == listID {
			doneAt := t.UpdatedAt
			if doneAt.IsZero() || !t.Done {
				doneAt = now
			}
			toArchive = append(toArchive, model.ArchivedCompletedTask{
				ID:           newID(),
				TaskText:     t.Text,
				OriginListID: list.ID,
				OriginList:   list.Name,
				Priority:     t.Priority,
				DoneAt:       doneAt,
				ArchivedAt:   now,
			})
			continue
		}
		kept = append(kept, t)
	}

	if len(toArchive) == 0 {
		return 0, ErrNoTasksInList
	}

	s.pushUndo()
	s.state.Tasks = kept
	s.normalizePositionsForList(listID)
	s.state.ArchivedCompleted = append(s.state.ArchivedCompleted, toArchive...)
	return len(toArchive), nil
}

func (s *Service) DeleteAllTasks(listID string) (int, error) {
	listID = strings.TrimSpace(listID)
	if listID == "" {
		return 0, ErrInvalidListRef
	}
	if !s.hasList(listID) {
		return 0, ErrListNotFound
	}

	kept := make([]model.Task, 0, len(s.state.Tasks))
	removed := 0
	for _, t := range s.state.Tasks {
		if t.ListID == listID {
			removed++
			continue
		}
		kept = append(kept, t)
	}
	if removed == 0 {
		return 0, ErrNoTasksInList
	}

	s.pushUndo()
	s.state.Tasks = kept
	s.normalizePositionsForList(listID)
	return removed, nil
}

func (s *Service) ArchivedCompleted() []model.ArchivedCompletedTask {
	out := make([]model.ArchivedCompletedTask, len(s.state.ArchivedCompleted))
	copy(out, s.state.ArchivedCompleted)
	return out
}

func (s *Service) SetSessionContext(activeListID, focus string) error {
	focus = strings.TrimSpace(focus)
	switch focus {
	case "", model.SessionFocusLists, model.SessionFocusTasks:
	default:
		return fmt.Errorf("%w: %s", ErrInvalidSessionFocus, focus)
	}
	if activeListID != "" && !s.hasList(activeListID) {
		activeListID = ""
	}
	if focus != "" {
		s.state.Metadata.Session.Focus = focus
	}
	s.state.Metadata.Session.ActiveListID = activeListID
	return nil
}

func (s *Service) MarkOnboardingSeen() {
	s.state.Metadata.FirstRun = false
}

func (s *Service) SetFilter(filter model.Filter) error {
	switch filter {
	case model.FilterAll, model.FilterTodo, model.FilterDone:
		s.state.Filter = filter
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidFilter, filter)
	}
}

func (s *Service) SetQuery(query string) {
	s.state.Query = strings.TrimSpace(query)
}

func (s *Service) FilteredTasks() []model.Task {
	q := strings.ToLower(strings.TrimSpace(s.state.Query))
	all := s.Tasks("")
	out := make([]model.Task, 0, len(all))
	for _, t := range all {
		if !matchesFilter(s.state.Filter, t.Done) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(t.Text), q) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// Undo reverts the latest mutable action from the undo stack.
func (s *Service) Undo() error {
	if len(s.undo) == 0 {
		return ErrNothingToUndo
	}
	last := s.undo[len(s.undo)-1]
	s.undo = s.undo[:len(s.undo)-1]
	s.state = copyState(last)
	return nil
}

// UndoDelete is kept for compatibility with older callers.
func (s *Service) UndoDelete() error {
	return s.Undo()
}

func (s *Service) hasList(listID string) bool {
	for _, l := range s.state.Lists {
		if l.ID == listID {
			return true
		}
	}
	return false
}

func (s *Service) nextTodoInsertPosition(listID string) int {
	firstDonePos := 0
	maxPos := 0

	for _, t := range s.state.Tasks {
		if t.ListID != listID {
			continue
		}
		if t.Position > maxPos {
			maxPos = t.Position
		}
		if t.Done && (firstDonePos == 0 || t.Position < firstDonePos) {
			firstDonePos = t.Position
		}
	}

	if firstDonePos > 0 {
		return firstDonePos
	}
	return maxPos + 1
}

func (s *Service) pushUndo() {
	s.undo = append(s.undo, copyState(s.state))
	if len(s.undo) > undoStackLimit {
		s.undo = s.undo[len(s.undo)-undoStackLimit:]
	}
}

func (s *Service) taskIndexesForList(listID string) []int {
	indexes := make([]int, 0)
	for i := range s.state.Tasks {
		if s.state.Tasks[i].ListID == listID {
			indexes = append(indexes, i)
		}
	}
	sort.SliceStable(indexes, func(i, j int) bool {
		a := s.state.Tasks[indexes[i]]
		b := s.state.Tasks[indexes[j]]
		if a.Position != b.Position {
			return a.Position < b.Position
		}
		return a.ID < b.ID
	})
	return indexes
}

func (s *Service) normalizePositionsForList(listID string) {
	indexes := s.taskIndexesForList(listID)
	for i, idx := range indexes {
		s.state.Tasks[idx].Position = i + 1
	}
}

func normalizeState(state model.AppState) model.AppState {
	if state.Lists == nil {
		state.Lists = []model.List{}
	}
	if state.Tasks == nil {
		state.Tasks = []model.Task{}
	}
	if state.ArchivedCompleted == nil {
		state.ArchivedCompleted = []model.ArchivedCompletedTask{}
	}
	if state.Filter == "" {
		state.Filter = model.FilterAll
	}
	if state.Metadata.Version == 0 {
		state.Metadata.Version = 1
	}
	if strings.TrimSpace(state.Metadata.Session.Focus) == "" {
		state.Metadata.Session.Focus = model.SessionFocusLists
	}
	if state.Metadata.Session.Focus != model.SessionFocusLists && state.Metadata.Session.Focus != model.SessionFocusTasks {
		state.Metadata.Session.Focus = model.SessionFocusLists
	}

	grouped := make(map[string][]int)
	for i := range state.Tasks {
		grouped[state.Tasks[i].ListID] = append(grouped[state.Tasks[i].ListID], i)
		if state.Tasks[i].Priority < model.PriorityNone || state.Tasks[i].Priority > model.PriorityHigh {
			state.Tasks[i].Priority = model.PriorityNone
		}
	}

	for i := range state.ArchivedCompleted {
		if state.ArchivedCompleted[i].Priority < model.PriorityNone || state.ArchivedCompleted[i].Priority > model.PriorityHigh {
			state.ArchivedCompleted[i].Priority = model.PriorityNone
		}
	}

	for _, indexes := range grouped {
		sort.SliceStable(indexes, func(i, j int) bool {
			a := state.Tasks[indexes[i]]
			b := state.Tasks[indexes[j]]

			aHasPos := a.Position > 0
			bHasPos := b.Position > 0
			if aHasPos != bHasPos {
				return aHasPos
			}
			if aHasPos && bHasPos && a.Position != b.Position {
				return a.Position < b.Position
			}
			return indexes[i] < indexes[j]
		})
		for order, idx := range indexes {
			state.Tasks[idx].Position = order + 1
		}
	}

	if state.Metadata.Session.ActiveListID != "" && !listIDExists(state.Lists, state.Metadata.Session.ActiveListID) {
		state.Metadata.Session.ActiveListID = ""
	}

	return state
}

func matchesFilter(filter model.Filter, done bool) bool {
	switch filter {
	case model.FilterDone:
		return done
	case model.FilterTodo:
		return !done
	default:
		return true
	}
}

func sortTasks(tasks []model.Task, singleListID string) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a := tasks[i]
		b := tasks[j]

		if singleListID == "" && a.ListID != b.ListID {
			return a.ListID < b.ListID
		}
		if a.Position != b.Position {
			return a.Position < b.Position
		}
		return a.ID < b.ID
	})
}

func copyState(state model.AppState) model.AppState {
	lists := make([]model.List, len(state.Lists))
	copy(lists, state.Lists)
	tasks := make([]model.Task, len(state.Tasks))
	copy(tasks, state.Tasks)
	archived := make([]model.ArchivedCompletedTask, len(state.ArchivedCompleted))
	copy(archived, state.ArchivedCompleted)

	out := state
	out.Lists = lists
	out.Tasks = tasks
	out.ArchivedCompleted = archived
	return out
}

func listIDExists(lists []model.List, id string) bool {
	for _, l := range lists {
		if l.ID == id {
			return true
		}
	}
	return false
}

func newID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(buf)
}
