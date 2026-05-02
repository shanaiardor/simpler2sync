package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type TaskDialogCallback func(task config.SyncTask)

func ShowTaskDialog(win fyne.Window, task *config.SyncTask, onSave TaskDialogCallback) {
	name := widget.NewEntry()
	name.SetPlaceHolder("Work notes")
	localPath := widget.NewEntry()
	localPath.SetPlaceHolder("Local folder path")
	bucket := widget.NewEntry()
	bucket.SetPlaceHolder("R2 bucket")
	prefix := widget.NewEntry()
	prefix.SetPlaceHolder("obsidian/")
	enabled := widget.NewCheck("Enabled", nil)

	if task != nil {
		name.SetText(task.Name)
		localPath.SetText(task.LocalPath)
		bucket.SetText(task.RemoteBucket)
		prefix.SetText(task.RemotePrefix)
		enabled.Checked = task.Enabled
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Local Path", localPath),
			widget.NewFormItem("R2 Bucket", bucket),
			widget.NewFormItem("R2 Prefix", prefix),
		},
	}
	content := container.NewVBox(
		widget.NewIcon(theme.FolderIcon()),
		form,
		enabled,
	)

	d := dialog.NewCustomConfirm("Sync Task", "Save", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		nt := config.SyncTask{
			Name:         name.Text,
			LocalPath:    localPath.Text,
			RemoteBucket: bucket.Text,
			RemotePrefix: prefix.Text,
			Enabled:      enabled.Checked,
		}
		onSave(nt)
	}, win)
	d.Resize(fyne.NewSize(520, 360))
	d.Show()
}
