package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func page(title, subtitle string, content fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	subtitleLabel := widget.NewLabel(subtitle)
	subtitleLabel.Wrapping = fyne.TextWrapWord

	return container.NewBorder(
		container.NewPadded(container.NewVBox(titleLabel, subtitleLabel, widget.NewSeparator())),
		nil,
		nil,
		nil,
		container.NewPadded(content),
	)
}

func section(title, subtitle string, content fyne.CanvasObject) fyne.CanvasObject {
	return widget.NewCard(title, subtitle, container.NewPadded(content))
}
