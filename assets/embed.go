package assets

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconData []byte

func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("icon.png", iconData)
}
