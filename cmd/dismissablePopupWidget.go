package cmd

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ShowDismissablePopup(window fyne.Window, msg string) {
	once := sync.Once{}

	var modal *widget.PopUp

	text := widget.NewMultiLineEntry()
	text.OnChanged = func(_ string) {
		text.SetText(msg)
	}
	text.SetText(msg)

	buttons := container.NewHBox(
		widget.NewButton("Copy Text", func() {
			window.Clipboard().SetContent(msg)
		}),
		widget.NewButton("Okay", func() {
			once.Do(modal.Hide)
		}),
	)

	priorityContainer := container.New(&priorityVLayout{}, text, container.NewCenter(buttons))

	modal = widget.NewModalPopUp(priorityContainer, window.Canvas())
	modal.Show()

	modal.Resize(fyne.NewSize(window.Canvas().Size().Width, modal.Size().Height/2))
}
