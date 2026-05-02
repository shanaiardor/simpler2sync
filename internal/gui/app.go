package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	"simpler2sync/internal/config"
)

type GUIApp struct {
	fyneApp fyne.App
	window  fyne.Window
	onQuit  func()

	cfg          *config.AppConfig
	taskList     *TaskList
	configPage   *ConfigPage
	logPage      *LogPage
	settingsPage *SettingsPage

	syncCallback func(task *config.SyncTask)
}

func NewGUIApp(name string, onQuit func()) *GUIApp {
	a := &GUIApp{
		fyneApp: app.NewWithID("simpler2sync"),
		onQuit:  onQuit,
	}
	a.fyneApp.SetIcon(AppIcon())
	return a
}

func (a *GUIApp) Init(cfg *config.AppConfig, syncCb func(task *config.SyncTask)) {
	a.cfg = cfg
	a.syncCallback = syncCb

	a.taskList = NewTaskList(cfg, func(action string, task *config.SyncTask) {
		switch action {
		case "add":
			ShowTaskDialog(a.window, nil, func(t config.SyncTask) {
				a.cfg.Tasks = append(a.cfg.Tasks, t)
				a.cfg.Save()
				a.taskList.Refresh()
			})
		case "edit":
			ShowTaskDialog(a.window, task, func(t config.SyncTask) {
				idx := -1
				for i, ts := range a.cfg.Tasks {
					if ts.Name == task.Name {
						idx = i
						break
					}
				}
				if idx >= 0 {
					a.cfg.Tasks[idx] = t
					a.cfg.Save()
					a.taskList.Refresh()
				}
			})
		case "sync":
			if a.syncCallback != nil {
				a.syncCallback(task)
			}
		}
	})
	a.configPage = NewConfigPage(cfg)
	a.logPage = NewLogPage()
	a.settingsPage = NewSettingsPage(cfg)
}

func (a *GUIApp) Run() {
	a.window = a.fyneApp.NewWindow("simpler2sync")
	a.window.SetMaster()
	a.window.Resize(fyne.NewSize(980, 680))

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Tasks", theme.ListIcon(), a.taskList.Content(a.window)),
		container.NewTabItemWithIcon("R2", theme.StorageIcon(), a.configPage.Content(a.window)),
		container.NewTabItemWithIcon("Log", theme.DocumentIcon(), a.logPage.Content()),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), a.settingsPage.Content(a.window)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	a.window.SetContent(tabs)

	if desk, ok := a.fyneApp.(desktop.App); ok {
		m := fyne.NewMenu("simpler2sync",
			fyne.NewMenuItem("Show", func() {
				a.window.Show()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				if a.onQuit != nil {
					a.onQuit()
				}
				a.fyneApp.Quit()
			}),
		)
		desk.SetSystemTrayMenu(m)
	}

	a.window.SetCloseIntercept(func() {
		a.window.Hide()
	})

	a.window.ShowAndRun()
}

func (a *GUIApp) Log(text string) {
	if a.logPage != nil {
		a.logPage.Append(text)
	}
}

func (a *GUIApp) RefreshTasks() {
	if a.taskList != nil {
		a.taskList.Refresh()
	}
}

func (a *GUIApp) Window() fyne.Window {
	return a.window
}

func (a *GUIApp) Quit() {
	a.fyneApp.Quit()
}
