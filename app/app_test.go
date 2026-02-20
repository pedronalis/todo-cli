package app

import (
	"errors"
	"testing"

	"todo-cli/model"
)

func mustCreateList(t *testing.T, svc *Service, name string) model.List {
	t.Helper()
	l, err := svc.CreateList(name, "")
	if err != nil {
		t.Fatalf("create list failed: %v", err)
	}
	return l
}

func mustCreateTask(t *testing.T, svc *Service, listID, text string) model.Task {
	t.Helper()
	tk, err := svc.CreateTask(listID, text)
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}
	return tk
}

func TestSetTaskPriorityValidation(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")
	task := mustCreateTask(t, svc, list.ID, "Pay bills")

	updated, err := svc.SetTaskPriority(task.ID, model.PriorityHigh)
	if err != nil {
		t.Fatalf("set priority failed: %v", err)
	}
	if updated.Priority != model.PriorityHigh {
		t.Fatalf("expected priority %d, got %d", model.PriorityHigh, updated.Priority)
	}

	if _, err := svc.SetTaskPriority(task.ID, 99); !errors.Is(err, ErrInvalidPriority) {
		t.Fatalf("expected ErrInvalidPriority, got %v", err)
	}

	if _, err := svc.SetTaskPriority("missing", model.PriorityLow); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestManualOrderingMoveWithLimits(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")
	a := mustCreateTask(t, svc, list.ID, "A")
	_ = mustCreateTask(t, svc, list.ID, "B")
	c := mustCreateTask(t, svc, list.ID, "C")

	if _, err := svc.MoveTaskUp(a.ID); !errors.Is(err, ErrTaskAlreadyAtTop) {
		t.Fatalf("expected ErrTaskAlreadyAtTop, got %v", err)
	}
	if _, err := svc.MoveTaskDown(c.ID); !errors.Is(err, ErrTaskAlreadyAtBottom) {
		t.Fatalf("expected ErrTaskAlreadyAtBottom, got %v", err)
	}

	if _, err := svc.MoveTaskDown(a.ID); err != nil {
		t.Fatalf("move down failed: %v", err)
	}
	tasks := svc.Tasks(list.ID)
	if tasks[0].Text != "B" || tasks[1].Text != "A" || tasks[2].Text != "C" {
		t.Fatalf("unexpected order after first move: %+v", tasks)
	}

	if _, err := svc.MoveTaskUp(c.ID); err != nil {
		t.Fatalf("move up failed: %v", err)
	}
	tasks = svc.Tasks(list.ID)
	if tasks[0].Text != "B" || tasks[1].Text != "C" || tasks[2].Text != "A" {
		t.Fatalf("unexpected order after second move: %+v", tasks)
	}
}

func TestToggleDoneMovesTaskToEnd(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")
	a := mustCreateTask(t, svc, list.ID, "A")
	b := mustCreateTask(t, svc, list.ID, "B")
	c := mustCreateTask(t, svc, list.ID, "C")

	if _, err := svc.ToggleDone(b.ID); err != nil {
		t.Fatalf("toggle done failed: %v", err)
	}
	ordered := svc.Tasks(list.ID)
	if ordered[0].ID != a.ID || ordered[1].ID != c.ID || ordered[2].ID != b.ID {
		t.Fatalf("expected done task to move to end, got order %+v", ordered)
	}
	if !ordered[2].Done {
		t.Fatalf("expected moved task to remain done")
	}
}

func TestCreateTaskInsertsBeforeDoneTasks(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")
	a := mustCreateTask(t, svc, list.ID, "A")
	b := mustCreateTask(t, svc, list.ID, "B")
	c := mustCreateTask(t, svc, list.ID, "C")

	if _, err := svc.ToggleDone(b.ID); err != nil {
		t.Fatalf("toggle done for B failed: %v", err)
	}
	if _, err := svc.ToggleDone(c.ID); err != nil {
		t.Fatalf("toggle done for C failed: %v", err)
	}

	newTask, err := svc.CreateTask(list.ID, "Nova")
	if err != nil {
		t.Fatalf("create task failed: %v", err)
	}

	ordered := svc.Tasks(list.ID)
	if len(ordered) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(ordered))
	}
	if ordered[0].ID != a.ID || ordered[1].ID != newTask.ID || ordered[2].ID != b.ID || ordered[3].ID != c.ID {
		t.Fatalf("expected new task before done tasks, got order %+v", ordered)
	}
	if ordered[1].Done {
		t.Fatalf("expected new task to be open")
	}
}

func TestSearchCombinedWithStatusFilter(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")
	todoMilk := mustCreateTask(t, svc, list.ID, "Buy milk")
	doneBook := mustCreateTask(t, svc, list.ID, "Read book")
	doneMilk := mustCreateTask(t, svc, list.ID, "Milkshake recipe")

	if _, err := svc.ToggleDone(doneBook.ID); err != nil {
		t.Fatalf("toggle doneBook failed: %v", err)
	}
	if _, err := svc.ToggleDone(doneMilk.ID); err != nil {
		t.Fatalf("toggle doneMilk failed: %v", err)
	}

	svc.SetQuery("milk")
	if err := svc.SetFilter(model.FilterAll); err != nil {
		t.Fatalf("set filter all failed: %v", err)
	}
	if got := len(svc.FilteredTasks()); got != 2 {
		t.Fatalf("expected 2 tasks for query+all, got %d", got)
	}

	if err := svc.SetFilter(model.FilterDone); err != nil {
		t.Fatalf("set filter done failed: %v", err)
	}
	done := svc.FilteredTasks()
	if len(done) != 1 || done[0].ID != doneMilk.ID {
		t.Fatalf("expected only done milk task, got %+v", done)
	}

	if err := svc.SetFilter(model.FilterTodo); err != nil {
		t.Fatalf("set filter todo failed: %v", err)
	}
	todo := svc.FilteredTasks()
	if len(todo) != 1 || todo[0].ID != todoMilk.ID {
		t.Fatalf("expected only todo milk task, got %+v", todo)
	}
}

func TestUndoRevertsAllSupportedMutableActions(t *testing.T) {
	svc := NewService(model.NewState())

	list, err := svc.CreateList("Inbox", "blue")
	if err != nil {
		t.Fatalf("create list failed: %v", err)
	}
	t1, err := svc.CreateTask(list.ID, "A")
	if err != nil {
		t.Fatalf("create t1 failed: %v", err)
	}
	t2, err := svc.CreateTask(list.ID, "B")
	if err != nil {
		t.Fatalf("create t2 failed: %v", err)
	}
	if _, err := svc.UpdateTask(t1.ID, "A1"); err != nil {
		t.Fatalf("update task failed: %v", err)
	}
	if _, err := svc.ToggleDone(t1.ID); err != nil {
		t.Fatalf("toggle done failed: %v", err)
	}
	if _, err := svc.SetTaskPriority(t1.ID, model.PriorityHigh); err != nil {
		t.Fatalf("set priority failed: %v", err)
	}
	if _, err := svc.MoveTaskDown(t2.ID); err != nil {
		t.Fatalf("move down failed: %v", err)
	}
	if err := svc.DeleteTask(t2.ID); err != nil {
		t.Fatalf("delete task failed: %v", err)
	}
	if _, err := svc.UpdateList(list.ID, "Inbox 2", "red"); err != nil {
		t.Fatalf("update list failed: %v", err)
	}
	if err := svc.DeleteList(list.ID); err != nil {
		t.Fatalf("delete list failed: %v", err)
	}

	if got := svc.State(); len(got.Lists) != 0 || len(got.Tasks) != 0 {
		t.Fatalf("expected empty state after delete list, got %+v", got)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo delete list failed: %v", err)
	}
	st := svc.State()
	if len(st.Lists) != 1 || len(st.Tasks) != 1 {
		t.Fatalf("expected 1 list + 1 task after undo delete list, got lists=%d tasks=%d", len(st.Lists), len(st.Tasks))
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo update list failed: %v", err)
	}
	st = svc.State()
	if st.Lists[0].Name != "Inbox" {
		t.Fatalf("expected list name restored to Inbox, got %q", st.Lists[0].Name)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo delete task failed: %v", err)
	}
	st = svc.State()
	if len(st.Tasks) != 2 {
		t.Fatalf("expected 2 tasks after undo delete task, got %d", len(st.Tasks))
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo move failed: %v", err)
	}
	ordered := svc.Tasks(list.ID)
	if ordered[0].ID != t2.ID {
		t.Fatalf("expected t2 back to first position after undo move, got order %+v", ordered)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo priority failed: %v", err)
	}
	restoredT1, _ := svc.GetTask(t1.ID)
	if restoredT1.Priority != model.PriorityNone {
		t.Fatalf("expected priority none after undo, got %d", restoredT1.Priority)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo toggle failed: %v", err)
	}
	restoredT1, _ = svc.GetTask(t1.ID)
	if restoredT1.Done {
		t.Fatalf("expected task not done after undo")
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo update task failed: %v", err)
	}
	restoredT1, _ = svc.GetTask(t1.ID)
	if restoredT1.Text != "A" {
		t.Fatalf("expected original text A after undo, got %q", restoredT1.Text)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo create t2 failed: %v", err)
	}
	if got := len(svc.State().Tasks); got != 1 {
		t.Fatalf("expected 1 task after undo create t2, got %d", got)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo create t1 failed: %v", err)
	}
	if got := len(svc.State().Tasks); got != 0 {
		t.Fatalf("expected 0 tasks after undo create t1, got %d", got)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo create list failed: %v", err)
	}
	if got := len(svc.State().Lists); got != 0 {
		t.Fatalf("expected 0 lists after undo create list, got %d", got)
	}

	if err := svc.Undo(); !errors.Is(err, ErrNothingToUndo) {
		t.Fatalf("expected ErrNothingToUndo after consuming stack, got %v", err)
	}
}

func TestUndoStackLimit20(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")

	for i := 0; i < 25; i++ {
		if _, err := svc.CreateTask(list.ID, "Task"); err != nil {
			t.Fatalf("create task %d failed: %v", i, err)
		}
	}

	for i := 0; i < 20; i++ {
		if err := svc.Undo(); err != nil {
			t.Fatalf("undo %d failed: %v", i, err)
		}
	}

	st := svc.State()
	if len(st.Lists) != 1 {
		t.Fatalf("expected list creation outside last 20 undos to remain, got %d lists", len(st.Lists))
	}
	if len(st.Tasks) != 5 {
		t.Fatalf("expected 5 tasks remaining after undoing capped stack, got %d", len(st.Tasks))
	}

	if err := svc.Undo(); !errors.Is(err, ErrNothingToUndo) {
		t.Fatalf("expected ErrNothingToUndo after consuming capped stack, got %v", err)
	}
}

func TestMoveListWithBounds(t *testing.T) {
	svc := NewService(model.NewState())
	a := mustCreateList(t, svc, "A")
	b := mustCreateList(t, svc, "B")
	c := mustCreateList(t, svc, "C")

	if _, err := svc.MoveListUp(a.ID); !errors.Is(err, ErrListAlreadyAtTop) {
		t.Fatalf("expected ErrListAlreadyAtTop, got %v", err)
	}
	if _, err := svc.MoveListDown(c.ID); !errors.Is(err, ErrListAlreadyAtBottom) {
		t.Fatalf("expected ErrListAlreadyAtBottom, got %v", err)
	}

	if _, err := svc.MoveListDown(a.ID); err != nil {
		t.Fatalf("move down failed: %v", err)
	}
	lists := svc.Lists()
	if lists[0].ID != b.ID || lists[1].ID != a.ID || lists[2].ID != c.ID {
		t.Fatalf("unexpected list order after move: %+v", lists)
	}
}

func TestClearCompletedToArchiveAndUndo(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")
	a := mustCreateTask(t, svc, list.ID, "A")
	b := mustCreateTask(t, svc, list.ID, "B")
	if _, err := svc.SetTaskPriority(b.ID, model.PriorityHigh); err != nil {
		t.Fatalf("set priority failed: %v", err)
	}
	if _, err := svc.ToggleDone(a.ID); err != nil {
		t.Fatalf("toggle A failed: %v", err)
	}
	if _, err := svc.ToggleDone(b.ID); err != nil {
		t.Fatalf("toggle B failed: %v", err)
	}

	moved, err := svc.ClearCompletedToArchive(list.ID)
	if err != nil {
		t.Fatalf("clear completed failed: %v", err)
	}
	if moved != 2 {
		t.Fatalf("expected 2 moved tasks, got %d", moved)
	}
	if got := len(svc.Tasks(list.ID)); got != 0 {
		t.Fatalf("expected 0 active tasks after archiving, got %d", got)
	}
	archived := svc.ArchivedCompleted()
	if len(archived) != 2 {
		t.Fatalf("expected 2 archived entries, got %d", len(archived))
	}
	if archived[0].OriginList != "Inbox" || archived[1].OriginList != "Inbox" {
		t.Fatalf("expected origin list metadata to be preserved")
	}
	if archived[1].Priority != model.PriorityHigh {
		t.Fatalf("expected archived high priority to be preserved, got %d", archived[1].Priority)
	}
	if archived[0].DoneAt.IsZero() || archived[0].ArchivedAt.IsZero() {
		t.Fatalf("expected timestamps in archived entry")
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo archive failed: %v", err)
	}
	if got := len(svc.Tasks(list.ID)); got != 2 {
		t.Fatalf("expected 2 tasks restored by undo, got %d", got)
	}
	if got := len(svc.ArchivedCompleted()); got != 0 {
		t.Fatalf("expected archived history reverted by undo, got %d", got)
	}

	if _, err := svc.ClearCompletedToArchive(list.ID); err != nil {
		t.Fatalf("archive second pass failed: %v", err)
	}
	if _, err := svc.ClearCompletedToArchive(list.ID); !errors.Is(err, ErrNoCompletedToClear) {
		t.Fatalf("expected ErrNoCompletedToClear, got %v", err)
	}
}

func TestArchiveAllToArchiveAndDeleteAllTasks(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")
	a := mustCreateTask(t, svc, list.ID, "A")
	b := mustCreateTask(t, svc, list.ID, "B")
	if _, err := svc.SetTaskPriority(b.ID, model.PriorityMedium); err != nil {
		t.Fatalf("set priority failed: %v", err)
	}
	if _, err := svc.ToggleDone(a.ID); err != nil {
		t.Fatalf("toggle done failed: %v", err)
	}

	moved, err := svc.ArchiveAllToArchive(list.ID)
	if err != nil {
		t.Fatalf("archive all failed: %v", err)
	}
	if moved != 2 {
		t.Fatalf("expected 2 archived tasks, got %d", moved)
	}
	if got := len(svc.Tasks(list.ID)); got != 0 {
		t.Fatalf("expected no active tasks after archive all, got %d", got)
	}
	archived := svc.ArchivedCompleted()
	if len(archived) != 2 {
		t.Fatalf("expected 2 archived history entries, got %d", len(archived))
	}
	if archived[1].Priority != model.PriorityMedium {
		t.Fatalf("expected priority to persist, got %d", archived[1].Priority)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo archive all failed: %v", err)
	}
	if got := len(svc.Tasks(list.ID)); got != 2 {
		t.Fatalf("expected tasks restored after undo, got %d", got)
	}

	deleted, err := svc.DeleteAllTasks(list.ID)
	if err != nil {
		t.Fatalf("delete all failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deletions, got %d", deleted)
	}
	if got := len(svc.Tasks(list.ID)); got != 0 {
		t.Fatalf("expected no tasks after delete all, got %d", got)
	}

	if err := svc.Undo(); err != nil {
		t.Fatalf("undo delete all failed: %v", err)
	}
	if got := len(svc.Tasks(list.ID)); got != 2 {
		t.Fatalf("expected tasks restored after undo delete all, got %d", got)
	}

	if _, err := svc.DeleteAllTasks("missing"); !errors.Is(err, ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got %v", err)
	}
	if _, err := svc.ArchiveAllToArchive(list.ID); err != nil {
		t.Fatalf("archive all second pass failed: %v", err)
	}
	if _, err := svc.ArchiveAllToArchive(list.ID); !errors.Is(err, ErrNoTasksInList) {
		t.Fatalf("expected ErrNoTasksInList, got %v", err)
	}
}

func TestSetSessionContextValidation(t *testing.T) {
	svc := NewService(model.NewState())
	list := mustCreateList(t, svc, "Inbox")

	if err := svc.SetSessionContext(list.ID, model.SessionFocusTasks); err != nil {
		t.Fatalf("set session context failed: %v", err)
	}
	st := svc.State()
	if st.Metadata.Session.ActiveListID != list.ID {
		t.Fatalf("expected active list in session metadata")
	}
	if st.Metadata.Session.Focus != model.SessionFocusTasks {
		t.Fatalf("expected session focus tasks, got %q", st.Metadata.Session.Focus)
	}

	if err := svc.SetSessionContext("", "bogus"); !errors.Is(err, ErrInvalidSessionFocus) {
		t.Fatalf("expected ErrInvalidSessionFocus, got %v", err)
	}
}
