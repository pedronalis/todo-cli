package model

import "time"

// Filter represents how tasks should be shown.
type Filter string

const (
	FilterAll  Filter = "all"
	FilterTodo Filter = "todo"
	FilterDone Filter = "done"
)

// Priority is a numeric task priority.
// 0=none, 1=low, 2=medium, 3=high.
type Priority int

const (
	PriorityNone   Priority = 0
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

// List is a task container/category.
type List struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Task is an individual todo item.
type Task struct {
	ID        string    `json:"id"`
	ListID    string    `json:"listId"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	Priority  Priority  `json:"priority,omitempty"`
	Position  int       `json:"position,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ArchivedCompletedTask keeps a historic record of completed items moved out of active list view.
type ArchivedCompletedTask struct {
	ID           string    `json:"id"`
	TaskText     string    `json:"taskText"`
	OriginListID string    `json:"originListId,omitempty"`
	OriginList   string    `json:"originList"`
	Priority     Priority  `json:"priority,omitempty"`
	DoneAt       time.Time `json:"doneAt"`
	ArchivedAt   time.Time `json:"archivedAt"`
}

const (
	SessionFocusLists = "lists"
	SessionFocusTasks = "tasks"
)

// SessionContext stores UI context that should survive restarts.
type SessionContext struct {
	ActiveListID string `json:"activeListId,omitempty"`
	Focus        string `json:"focus,omitempty"`
}

// Metadata is app-level metadata persisted alongside the state.
type Metadata struct {
	Version  int            `json:"version"`
	FirstRun bool           `json:"firstRun,omitempty"`
	Session  SessionContext `json:"session,omitempty"`
}

// AppState is the full persisted state.
type AppState struct {
	Lists             []List                  `json:"lists"`
	Tasks             []Task                  `json:"tasks"`
	ArchivedCompleted []ArchivedCompletedTask `json:"archivedCompleted,omitempty"`
	Filter            Filter                  `json:"filter,omitempty"`
	Query             string                  `json:"query,omitempty"`
	Metadata          Metadata                `json:"metadata,omitempty"`
}

// NewState returns an initialized empty state.
func NewState() AppState {
	return AppState{
		Lists:             []List{},
		Tasks:             []Task{},
		ArchivedCompleted: []ArchivedCompletedTask{},
		Filter:            FilterAll,
		Query:             "",
		Metadata: Metadata{
			Version:  1,
			FirstRun: true,
			Session: SessionContext{
				Focus: SessionFocusLists,
			},
		},
	}
}
