package tui

import (
	"testing"

	"todo-cli/app"
	"todo-cli/model"
)

func TestPaneWidthsPreferNarrowLeftPanel(t *testing.T) {
	svc := app.NewService(model.NewState())
	m := NewModel(svc, "", "")
	m.width = 120

	viewW := m.viewportWidth()
	left, right := m.paneWidths(viewW, 1)
	if left >= right {
		t.Fatalf("expected left panel to be narrower than right (left=%d right=%d)", left, right)
	}
	if left+right+1 != viewW {
		t.Fatalf("expected pane widths to fill available width=%d, got left=%d right=%d", viewW, left, right)
	}
}

func TestPaneWidthsSmallTerminalStillValid(t *testing.T) {
	svc := app.NewService(model.NewState())
	m := NewModel(svc, "", "")
	m.width = 48

	viewW := m.viewportWidth()
	left, right := m.paneWidths(viewW, 1)
	if left < 10 || right < 12 {
		t.Fatalf("expected minimum usable pane widths, got left=%d right=%d", left, right)
	}
	if left+right+1 > viewW {
		t.Fatalf("expected panes not to exceed viewport width=%d, got left=%d right=%d", viewW, left, right)
	}
}
