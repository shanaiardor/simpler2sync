package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type TaskListCallback func(action string, task *config.SyncTask)

type TaskList struct {
	cfg        *config.AppConfig
	list       *widget.List
	selectedID int
	onAction   TaskListCallback
}

func NewTaskList(cfg *config.AppConfig, onAction TaskListCallback) *TaskList {
	tl := &TaskList{
		cfg:        cfg,
		selectedID: -1,
		onAction:   onAction,
	}
	tl.list = widget.NewList(
		func() int { return len(cfg.Tasks) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("Task")
			name.TextStyle = fyne.TextStyle{Bold: true}
			target := widget.NewLabel("bucket/prefix")
			local := widget.NewLabel("local path")
			local.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(name, target, local)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			t := cfg.Tasks[id]
			tl.list.SetItemHeight(id, 82)
			status := "Paused"
			if t.Enabled {
				status = "Active"
			}
			row := obj.(*fyne.Container)
			row.Objects[0].(*widget.Label).SetText(fmt.Sprintf("%s  %s", status, t.Name))
			row.Objects[1].(*widget.Label).SetText(fmt.Sprintf("R2  %s/%s", t.RemoteBucket, t.RemotePrefix))
			row.Objects[2].(*widget.Label).SetText(fmt.Sprintf("Local  %s", t.LocalPath))
		},
	)
	tl.list.OnSelected = func(id widget.ListItemID) {
		tl.selectedID = id
	}
	tl.list.OnUnselected = func(id widget.ListItemID) {
		tl.selectedID = -1
	}
	return tl
}

func (tl *TaskList) Content(win fyne.Window) fyne.CanvasObject {
	addBtn := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		tl.onAction("add", nil)
	})
	editBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		idx := tl.selectedID
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		task := tl.cfg.Tasks[idx]
		tl.onAction("edit", &task)
	})
	removeBtn := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		idx := tl.selectedID
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		tl.cfg.Tasks = append(tl.cfg.Tasks[:idx], tl.cfg.Tasks[idx+1:]...)
		tl.cfg.Save()
		tl.list.Refresh()
	})
	toggleBtn := widget.NewButtonWithIcon("Toggle", theme.MediaReplayIcon(), func() {
		idx := tl.selectedID
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		tl.cfg.Tasks[idx].Enabled = !tl.cfg.Tasks[idx].Enabled
		tl.cfg.Save()
		tl.list.Refresh()
	})
	syncBtn := widget.NewButtonWithIcon("Sync Now", theme.ViewRefreshIcon(), func() {
		idx := tl.selectedID
		if idx < 0 || idx >= len(tl.cfg.Tasks) {
			return
		}
		task := tl.cfg.Tasks[idx]
		tl.onAction("sync", &task)
	})
	syncBtn.Importance = widget.HighImportance

	btns := container.NewHBox(addBtn, editBtn, removeBtn, toggleBtn, syncBtn)
	body := container.NewBorder(nil, btns, nil, nil, tl.list)
	return page("Sync Tasks", "Manage folder pairs and run manual syncs.", body)
}

func (tl *TaskList) Refresh() {
	tl.list.Refresh()
}
