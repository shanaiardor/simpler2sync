package gui

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type LogPage struct {
	mu  sync.Mutex
	log *widget.Entry
}

func NewLogPage() *LogPage {
	lp := &LogPage{
		log: widget.NewMultiLineEntry(),
	}
	lp.log.SetPlaceHolder("Sync activity will appear here.")
	lp.log.Disable()
	return lp
}

func (lp *LogPage) Content() fyne.CanvasObject {
	clearBtn := widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), func() {
		lp.mu.Lock()
		lp.log.SetText("")
		lp.mu.Unlock()
	})
	body := container.NewBorder(nil, container.NewHBox(clearBtn), nil, nil, lp.log)
	return page("Activity Log", "Review sync progress, errors, and scheduled runs.", body)
}

func (lp *LogPage) Append(text string) {
	fyne.DoAndWait(func() {
		lp.mu.Lock()
		defer lp.mu.Unlock()
		lp.log.SetText(lp.log.Text + "\n" + text)
	})
}
