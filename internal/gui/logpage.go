package gui

import (
	"bytes"
	"runtime"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type LogPage struct {
	mu     sync.Mutex
	log    *widget.RichText
	scroll *container.Scroll
	lines  []string
}

func NewLogPage() *LogPage {
	lp := &LogPage{
		log: widget.NewRichText(),
	}
	lp.log.Wrapping = fyne.TextWrapWord
	lp.scroll = container.NewVScroll(lp.log)
	return lp
}

func (lp *LogPage) Content() fyne.CanvasObject {
	clearBtn := widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), func() {
		lp.mu.Lock()
		lp.lines = nil
		lp.log.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: "Sync activity will appear here."},
		}
		lp.log.Refresh()
		lp.mu.Unlock()
	})
	lp.log.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: "Sync activity will appear here."},
	}
	body := container.NewBorder(nil, container.NewHBox(clearBtn), nil, nil, lp.scroll)
	return page("Activity Log", "Review sync progress, errors, and scheduled runs.", body)
}

func (lp *LogPage) Append(text string) {
	update := func() {
		lp.mu.Lock()
		defer lp.mu.Unlock()
		lp.lines = append(lp.lines, text)
		lp.log.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: strings.Join(lp.lines, "\n")},
		}
		lp.log.Refresh()
		lp.scroll.ScrollToBottom()
	}
	if isMainGoroutine() {
		update()
		return
	}
	fyne.Do(update)
}

func isMainGoroutine() bool {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	return bytes.HasPrefix(buf[:n], []byte("goroutine 1 "))
}
