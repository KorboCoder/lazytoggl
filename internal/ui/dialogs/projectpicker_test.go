package dialogs

import "testing"

func feedPicker(p ProjectPicker, keys ...string) (ProjectPicker, PickResult) {
	res := PickEditing
	for _, k := range keys {
		p, res = p.Update(keyMsg(k))
	}
	return p, res
}

func TestProjectPickerPicks(t *testing.T) {
	p := NewProjectPicker(listProjects, false)
	p, res := feedPicker(p, "i", "n", "t", "enter")
	if res != PickPicked {
		t.Fatalf("result = %v, want picked", res)
	}
	if o := p.Picked(); o.Kind != OptionProject || o.Project.Name != "Internal" {
		t.Fatalf("picked = %+v, want Internal", o)
	}
}

func TestProjectPickerMovesSelection(t *testing.T) {
	p := NewProjectPicker(listProjects, true) // No project, ACME, acme website, Internal
	p, _ = feedPicker(p, "down", "ctrl+n", "up", "ctrl+n", "ctrl+n", "enter")
	if o := p.Picked(); o.Project.Name != "Internal" {
		t.Fatalf("picked = %+v, want Internal", o)
	}
	q, _ := feedPicker(NewProjectPicker(listProjects, true), "ctrl+n", "ctrl+p", "enter")
	if o := q.Picked(); o.Kind != OptionNone {
		t.Fatalf("picked = %+v, want No project", o)
	}
}

func TestProjectPickerTabPicks(t *testing.T) {
	_, res := feedPicker(NewProjectPicker(listProjects, false), "tab")
	if res != PickPicked {
		t.Fatalf("tab = %v, want picked", res)
	}
}

func TestProjectPickerEscCancels(t *testing.T) {
	_, res := feedPicker(NewProjectPicker(listProjects, false), "down", "esc")
	if res != PickCancelled {
		t.Fatalf("esc = %v, want cancelled", res)
	}
}

func TestProjectPickerCreateOption(t *testing.T) {
	p, _ := feedPicker(NewProjectPicker(listProjects, false), "n", "e", "w", "enter")
	if o := p.Picked(); o.Kind != OptionCreate || o.Name != "new" {
		t.Fatalf("picked = %+v, want create new", o)
	}
}

// A search matching only an archived name leaves nothing to pick: enter
// does nothing rather than picking a phantom option.
func TestProjectPickerEnterOnEmptyListStays(t *testing.T) {
	_, res := feedPicker(NewProjectPicker(listProjects, false), "o", "l", "d", " ", "c", "l", "i", "e", "n", "t", "enter")
	if res != PickEditing {
		t.Fatalf("enter on empty list = %v, want still editing", res)
	}
}
