package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"simpler2sync/internal/config"
)

type ConfigPage struct {
	cfg *config.AppConfig
}

func NewConfigPage(cfg *config.AppConfig) *ConfigPage {
	return &ConfigPage{cfg: cfg}
}

func (cp *ConfigPage) Content(win fyne.Window) fyne.CanvasObject {
	endpoint := widget.NewEntry()
	endpoint.SetPlaceHolder("https://<account_id>.r2.cloudflarestorage.com")
	endpoint.SetText(cp.cfg.R2.Endpoint)
	accessKey := widget.NewEntry()
	accessKey.SetPlaceHolder("Access key ID")
	accessKey.SetText(cp.cfg.R2.AccessKeyID)
	secretKey := widget.NewPasswordEntry()
	secretKey.SetPlaceHolder("Secret access key")
	secretKey.SetText(cp.cfg.R2.SecretAccessKey)
	region := widget.NewEntry()
	region.SetPlaceHolder("auto")
	region.SetText(cp.cfg.R2.Region)

	saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		cp.cfg.R2.Endpoint = endpoint.Text
		cp.cfg.R2.AccessKeyID = accessKey.Text
		cp.cfg.R2.SecretAccessKey = secretKey.Text
		cp.cfg.R2.Region = region.Text
		cp.cfg.R2.Type = "s3"
		cp.cfg.R2.Provider = "Cloudflare"
		cp.cfg.R2.ACL = "private"
		if err := cp.cfg.Save(); err != nil {
			dialog.ShowError(err, win)
		} else {
			dialog.ShowInformation("Success", "Configuration saved", win)
		}
	})
	saveBtn.Importance = widget.HighImportance

	loadBtn := widget.NewButtonWithIcon("Load rclone", theme.FolderOpenIcon(), func() {
		dialog.ShowFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			defer uc.Close()
			r2, err := config.LoadR2FromINIPath(uc.URI().Path())
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			endpoint.SetText(r2.Endpoint)
			accessKey.SetText(r2.AccessKeyID)
			secretKey.SetText(r2.SecretAccessKey)
			region.SetText(r2.Region)
		}, win)
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Endpoint", endpoint),
			widget.NewFormItem("Access Key ID", accessKey),
			widget.NewFormItem("Secret Access Key", secretKey),
			widget.NewFormItem("Region", region),
		},
	}

	btns := container.NewHBox(saveBtn, loadBtn)
	body := container.NewBorder(nil, btns, nil, nil, section("Cloudflare R2", "S3-compatible credentials used by all sync tasks.", form))
	return page("R2 Configuration", "Keep these credentials local and private.", body)
}
