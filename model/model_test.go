package model

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestAppStateSerializationRoundTrip(t *testing.T) {
	now := time.Date(2026, 2, 19, 12, 0, 0, 0, time.UTC)
	state := AppState{
		Lists: []List{
			{
				ID:        "l1",
				Name:      "Inbox",
				Color:     "blue",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Tasks: []Task{
			{
				ID:        "t1",
				ListID:    "l1",
				Text:      "write tests",
				Done:      true,
				Priority:  PriorityHigh,
				Position:  3,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		ArchivedCompleted: []ArchivedCompletedTask{
			{
				ID:           "a1",
				TaskText:     "old done",
				OriginListID: "l1",
				OriginList:   "Inbox",
				Priority:     PriorityMedium,
				DoneAt:       now,
				ArchivedAt:   now,
			},
		},
		Filter: FilterDone,
		Query:  "tests",
		Metadata: Metadata{
			Version:  1,
			FirstRun: false,
			Session: SessionContext{
				ActiveListID: "l1",
				Focus:        SessionFocusTasks,
			},
		},
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var got AppState
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(state, got) {
		t.Fatalf("round-trip mismatch\nwant=%+v\ngot=%+v", state, got)
	}
}
