package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// handleError on window with err and call f after user hits "Okay" button.
func handleError(w fyne.Window, err error, okay func()) {
	once := sync.Once{}

	w.SetContent(
		container.NewVBox(
			widget.NewLabel(err.Error()),
			widget.NewButton("Copy Error", func() {
				w.Clipboard().SetContent(err.Error())
			}),
			widget.NewButton("Okay", func() {
				once.Do(okay)
			}),
		),
	)
}

func login(w fyne.Window) (sess *filebrowserSession, err error) {
	done := make(chan struct{})

	hEntry := widget.NewEntry()
	hEntry.Text = config.Host
	uEntry := widget.NewEntry()
	uEntry.Text = config.User
	pEntry := widget.NewPasswordEntry()
	pEntry.Text = config.Pass

	form := &widget.Form{
		Items: []*widget.FormItem{
			{
				Text:   "Host",
				Widget: hEntry,
			},
			{
				Text:   "Username",
				Widget: uEntry,
			},
			{
				Text:   "Password",
				Widget: pEntry,
			},
		},
		OnSubmit: sync.OnceFunc(func() { close(done) }),
	}

	vbox := (container.NewVBox(layout.NewSpacer(), form, layout.NewSpacer()))
	w.SetContent(container.NewGridWithColumns(3, layout.NewSpacer(), vbox, layout.NewSpacer()))

	<-done

	w.SetContent(container.NewCenter(widget.NewLabel("Logging in")))

	if hEntry.Text != config.Host || uEntry.Text != config.User || pEntry.Text != config.Pass {
		config.Host = hEntry.Text
		config.User = uEntry.Text
		config.Pass = pEntry.Text
		config.changed = true
	}

	sess, err = loginToFilebrowser(config.Host, config.User, config.Pass)
	if err != nil {
		return nil, fmt.Errorf("could not login to (%v): %w", config.Host, err)
	}

	return sess, nil
}

func logic(w fyne.Window) {
	if config == nil { // TODO: use config.loaded?
		err := parseConfig()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				slog.Info("configuration file doesn't exist, instead using defaults", "error", err)

				// set loaded to true as if the file doesn't exist, we "successfully"
				// loaded the defaults. this lets things like the save config logic work
				config = &Config{loaded: true}

				go logic(w)

				return
			}

			handleError(w, fmt.Errorf("WARNING configuration file error: %w", err), func() {
				go logic(w)
			})
			return
		}
	}

	// TODO: this flow should be async, not required to sync back to this function every run
	// lock user into login loop until they login successfully.
	var (
		sess *filebrowserSession
		err  error
	)
	for {
		sess, err = login(w)
		if err != nil {
			acked := make(chan struct{})
			handleError(w, err, func() { close(acked) })
			<-acked
			continue
		}
		break
	}

	if config.changed {
		if config.loaded {
			if err := saveConfig(); err != nil {
				acked := make(chan struct{})
				handleError(w, fmt.Errorf("could not save config file: %w", err), func() { acked <- struct{}{} })
				<-acked
			}
		} else {
			continueNoSave := make(chan struct{})
			overwrite := make(chan struct{})

			w.SetContent(
				container.NewVBox(
					widget.NewLabel("Would you like to save over the current configuration file, even though it failed to load?\n\n"+
						"config file path: "+filepath.Join(config.Dir, configFileName)),
					widget.NewButton("Continue without saving", sync.OnceFunc(func() { close(continueNoSave) })),
					widget.NewButton("Overwrite config file", sync.OnceFunc(func() { close(overwrite) })),
				),
			)

			select {
			case <-overwrite:
				if err := saveConfig(); err != nil {
					acked := make(chan struct{})
					handleError(w, fmt.Errorf("could not save config file: %w", err), func() { acked <- struct{}{} })
					<-acked
				}
			case <-continueNoSave:
			}
		}
	}

	browse(w, sess)
}

func run() (err error) {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Info("Set loglevel", "level", "DEBUG")
	case "info":
		slog.SetLogLoggerLevel(slog.LevelInfo)
		slog.Info("Set log level", "level", "INFO")
	case "warn":
		slog.SetLogLoggerLevel(slog.LevelWarn)
		slog.Info("Set log level", "level", "WARN")
	case "error":
		slog.SetLogLoggerLevel(slog.LevelError)
		slog.Info("Set log level", "level", "ERROR")
	case "":
		slog.Info("Using default log level")
	default:
		slog.Error("unknown log level", "level", os.Getenv("LOG_LEVEL"))
	}

	a := app.New()
	w := a.NewWindow("FilebrowserUI")
	w.Resize(fyne.NewSize(700, 400))

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
