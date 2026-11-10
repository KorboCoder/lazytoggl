package dialogs

import (
	"testing"
	"time"

	"github.com/KorboCoder/lazytoggl/internal/toggl"
)

// Going back from the review onto the description row must restart the
// input's cursor blink, which stops while the review swallows its ticks.
func TestEditBackFromReviewRestartsDescriptionBlink(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.Local)
	e := toggl.TimeEntry{ID: 3, Description: "Task", Start: now.Add(-2 * time.Hour), Stop: now.Add(-time.Hour), Duration: 3600}
	var dlg Dialog = NewEdit(e, now, nil)
	dlg, _ = dlg.Update(keyMsg("X"))    // edit the description
	dlg, _ = dlg.Update(keyMsg("esc"))  // open the review
	_, cmd := dlg.Update(keyMsg("esc")) // back to the form
	if cmd == nil {
		t.Fatal("back from review returned no command; the description cursor stays frozen")
	}
}
