package gui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type SettingsPage struct {
	cfg *config.AppConfig
}

func NewSettingsPage(cfg *config.AppConfig) *SettingsPage {
	return &SettingsPage{cfg: cfg}
}

func (sp *SettingsPage) Content(win fyne.Window) fyne.CanvasObject {
	interval := widget.NewEntry()
	interval.SetPlaceHolder("300")
	interval.SetText(strconv.Itoa(sp.cfg.Settings.IntervalSeconds))
	cronExpr := widget.NewEntry()
	cronExpr.SetPlaceHolder("0 2 * * *")
	cronExpr.SetText(sp.cfg.Settings.CronExpression)
	concurrent := widget.NewEntry()
	concurrent.SetPlaceHolder("3")
	concurrent.SetText(strconv.Itoa(sp.cfg.Settings.ConcurrentTransfers))

	conflict := widget.NewSelect([]string{"newer", "mirror"}, func(v string) {
		sp.cfg.Settings.ConflictStrategy = v
	})
	if sp.cfg.Settings.ConflictStrategy != "" {
		conflict.SetSelected(sp.cfg.Settings.ConflictStrategy)
	} else {
		conflict.SetSelected("newer")
	}

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		if v, err := strconv.Atoi(interval.Text); err == nil {
			sp.cfg.Settings.IntervalSeconds = v
		}
		sp.cfg.Settings.CronExpression = cronExpr.Text
		if v, err := strconv.Atoi(concurrent.Text); err == nil {
			sp.cfg.Settings.ConcurrentTransfers = v
		}
		if err := sp.cfg.Save(); err != nil {
			dialog.ShowError(err, win)
		} else {
			dialog.ShowInformation("Success", "Settings saved", win)
		}
	})
	saveBtn.Importance = widget.HighImportance

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Interval (seconds)", interval),
			widget.NewFormItem("Cron Expression", cronExpr),
			widget.NewFormItem("Conflict Strategy", conflict),
			widget.NewFormItem("Concurrent Transfers", concurrent),
		},
	}
	body := container.NewBorder(nil, container.NewHBox(saveBtn), nil, nil, section("Scheduler", "Interval is used when cron is empty.", form))
	return page("Settings", "Tune background sync behavior and conflict handling.", body)
}
