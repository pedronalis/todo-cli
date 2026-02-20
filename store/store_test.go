package store

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"todo-cli/model"
)

func sampleState(label string) model.AppState {
	now := time.Date(2026, 2, 19, 12, 30, 0, 0, time.UTC)
	listID := "list-" + label
	taskID := "task-" + label
	return model.AppState{
		Lists: []model.List{{
			ID:        listID,
			Name:      "Inbox-" + label,
			Color:     "green",
			CreatedAt: now,
			UpdatedAt: now,
		}},
		Tasks: []model.Task{{
			ID:        taskID,
			ListID:    listID,
			Text:      "Task-" + label,
			Done:      false,
			Priority:  model.PriorityMedium,
			Position:  1,
			CreatedAt: now,
			UpdatedAt: now,
		}},
		ArchivedCompleted: []model.ArchivedCompletedTask{{
			ID:           "arch-" + label,
			TaskText:     "Done-" + label,
			OriginListID: listID,
			OriginList:   "Inbox-" + label,
			Priority:     model.PriorityLow,
			DoneAt:       now,
			ArchivedAt:   now,
		}},
		Filter: model.FilterAll,
		Query:  "Task",
		Metadata: model.Metadata{
			Version:  1,
			FirstRun: false,
			Session: model.SessionContext{
				ActiveListID: listID,
				Focus:        model.SessionFocusTasks,
			},
		},
	}
}

func TestLoadMissingFileReturnsEmptyState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	state, err := Load(path)
	if err != nil {
		t.Fatalf("load missing file failed: %v", err)
	}

	want := model.NewState()
	if !reflect.DeepEqual(want, state) {
		t.Fatalf("unexpected state for missing file\nwant=%+v\ngot=%+v", want, state)
	}
}

func TestSaveThenLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	want := sampleState("a")

	if err := Save(path, want); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("save/load mismatch\nwant=%+v\ngot=%+v", want, got)
	}
}

func TestAutosaveCreatesBackupAndPersistsLatestState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	initial := sampleState("old")
	updated := sampleState("new")

	if err := Save(path, initial); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}
	if err := Autosave(path, updated); err != nil {
		t.Fatalf("autosave failed: %v", err)
	}

	gotLatest, err := Load(path)
	if err != nil {
		t.Fatalf("load latest failed: %v", err)
	}
	if !reflect.DeepEqual(updated, gotLatest) {
		t.Fatalf("latest state mismatch\nwant=%+v\ngot=%+v", updated, gotLatest)
	}

	gotBackup, err := Load(path + ".bak")
	if err != nil {
		t.Fatalf("load backup failed: %v", err)
	}
	if !reflect.DeepEqual(initial, gotBackup) {
		t.Fatalf("backup mismatch\nwant=%+v\ngot=%+v", initial, gotBackup)
	}
}

func TestAutosaveRotatingBackupsArePruned(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := Save(path, sampleState("seed")); err != nil {
		t.Fatalf("seed save failed: %v", err)
	}

	for i := 0; i < 15; i++ {
		if err := Autosave(path, sampleState(fmt.Sprintf("%d", i))); err != nil {
			t.Fatalf("autosave %d failed: %v", i, err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	files, err := filepath.Glob(path + ".bak.*")
	if err != nil {
		t.Fatalf("glob rotating backups failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected rotating backups, found none")
	}
	if len(files) > maxRotatingBackups {
		t.Fatalf("expected at most %d rotating backups, got %d", maxRotatingBackups, len(files))
	}
}

func TestLoadWithRecoveryRestoresFromBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	v1 := sampleState("v1")
	v2 := sampleState("v2")
	v3 := sampleState("v3")

	if err := Save(path, v1); err != nil {
		t.Fatalf("save v1 failed: %v", err)
	}
	if err := Autosave(path, v2); err != nil {
		t.Fatalf("autosave v2 failed: %v", err)
	}
	if err := Autosave(path, v3); err != nil {
		t.Fatalf("autosave v3 failed: %v", err)
	}

	if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("corrupt write failed: %v", err)
	}

	recovered, status, err := LoadWithRecovery(path)
	if err != nil {
		t.Fatalf("load with recovery failed: %v", err)
	}
	if status == "" {
		t.Fatalf("expected recovery status message, got empty")
	}
	if !reflect.DeepEqual(v2, recovered) {
		t.Fatalf("expected recovery from latest backup (v2), got %+v", recovered)
	}

	persisted, err := Load(path)
	if err != nil {
		t.Fatalf("load persisted recovered state failed: %v", err)
	}
	if !reflect.DeepEqual(v2, persisted) {
		t.Fatalf("expected persisted recovered state to match v2")
	}

	corruptFiles, err := filepath.Glob(filepath.Join(dir, "state.corrupt-*.json"))
	if err != nil {
		t.Fatalf("glob corrupt files failed: %v", err)
	}
	if len(corruptFiles) != 1 {
		t.Fatalf("expected exactly one moved corrupt file, got %d", len(corruptFiles))
	}
}

func TestLoadWithRecoveryWithoutBackupStartsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("write corrupt state failed: %v", err)
	}

	recovered, status, err := LoadWithRecovery(path)
	if err != nil {
		t.Fatalf("load with recovery failed: %v", err)
	}
	if status == "" {
		t.Fatalf("expected recovery status message")
	}
	if !reflect.DeepEqual(model.NewState(), recovered) {
		t.Fatalf("expected empty state when no valid backup")
	}

	persisted, err := Load(path)
	if err != nil {
		t.Fatalf("load persisted empty state failed: %v", err)
	}
	if !reflect.DeepEqual(model.NewState(), persisted) {
		t.Fatalf("expected persisted empty state after recovery")
	}
}

func TestLoadLegacyJSONWithoutNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")
	legacy := `{
  "lists": [
    {
      "id": "l1",
      "name": "Inbox",
      "color": "blue",
      "createdAt": "2026-02-19T12:00:00Z",
      "updatedAt": "2026-02-19T12:00:00Z"
    }
  ],
  "tasks": [
    {
      "id": "t1",
      "listId": "l1",
      "text": "legacy task",
      "done": false,
      "createdAt": "2026-02-19T12:01:00Z",
      "updatedAt": "2026-02-19T12:01:00Z"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy file failed: %v", err)
	}

	state, err := Load(path)
	if err != nil {
		t.Fatalf("load legacy state failed: %v", err)
	}

	if state.Filter != model.FilterAll {
		t.Fatalf("expected default filter all, got %q", state.Filter)
	}
	if state.Metadata.Version != 1 {
		t.Fatalf("expected default metadata version 1, got %d", state.Metadata.Version)
	}
	if state.Metadata.Session.Focus != model.SessionFocusLists {
		t.Fatalf("expected default session focus lists, got %q", state.Metadata.Session.Focus)
	}
	if len(state.ArchivedCompleted) != 0 {
		t.Fatalf("expected empty archived history for legacy state")
	}
	if len(state.Tasks) != 1 {
		t.Fatalf("expected one legacy task, got %d", len(state.Tasks))
	}
	if state.Tasks[0].Priority != model.PriorityNone {
		t.Fatalf("expected default priority none, got %d", state.Tasks[0].Priority)
	}
	if state.Tasks[0].Position != 0 {
		t.Fatalf("expected default position 0 from legacy JSON, got %d", state.Tasks[0].Position)
	}
}
