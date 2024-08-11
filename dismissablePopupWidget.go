package main

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowDismissablePopup(window fyne.Window, msg string) {
	once := sync.Once{}

	var modal *widget.PopUp

	label := widget.NewLabel(msg)
	label.Alignment = fyne.TextAlignCenter
	scroll := container.NewHScroll(label)

	msgWidget := container.NewVBox(
		scroll,
		widget.NewButton("Copy Text", func() {
			window.Clipboard().SetContent(msg)
		}),
		widget.NewButton("Okay", func() {
			once.Do(modal.Hide)
		}),
	)

	modal = widget.NewModalPopUp(msgWidget, window.Canvas())
	modal.Show()

	modal.Resize(fyne.NewSize(window.Canvas().Size().Width, modal.Size().Height/2))
}
