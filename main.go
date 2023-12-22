package main

import (
	"fmt"
	"image/color"
	"log"
	"os"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/container"
	"fyne.io/fyne/widget"
)

func handleLogin(w fyne.Window) (user, pass string, err error) {
	done := make(chan struct{}, 1)

	uEntry := widget.NewEntry()
	pEntry := widget.NewEntry()
	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:   "Username",
				Widget: uEntry,
			},
			{
				Text:   "Password",
				Widget: pEntry,
			},
		},
		OnSubmit: func() {
			user = uEntry.Text
			pass = pEntry.Text

			select {
			case done <- struct{}{}:
			default:
			}
		},
	}

	content := container.NewBorder(canvas.NewText("asdfsa", color.White), nil, nil, nil, form)
	w.SetContent(content)

	<-done
	return user, pass, nil
}

func logic(w fyne.Window) (err error) {
	user, pass, err := handleLogin(w)
	if err != nil {
		return err
	}

	w.SetContent(widget.NewLabel(fmt.Sprintf("user: %v pass: %v", user, pass)))
	return nil
}

func run() (err error) {
	a := app.New()
	w := a.NewWindow("FilebrowserUI")
	w.Resize(fyne.NewSize(1280, 720))

	go logic(w)

	w.ShowAndRun()

	return nil
}

func main() {
	if err := run(); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
