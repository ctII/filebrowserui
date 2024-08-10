package main

import (
	"log/slog"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type nodeWidget struct {
	widget.BaseWidget

	checksumButton         *widget.Button
	checksumButtonFunc     func()
	checksumButtonFuncLock sync.RWMutex

	filenameLabel *widget.Label
}

var _ fyne.Widget = &nodeWidget{}

func NewNodeWidget() *nodeWidget {
	nw := &nodeWidget{}
	nw.ExtendBaseWidget(nw)

	nw.checksumButton = widget.NewButton("Checksum", func() {
		nw.checksumButtonFuncLock.RLock()
		defer nw.checksumButtonFuncLock.RUnlock()
		if nw.checksumButtonFunc != nil {
			slog.Debug("calling checksum button function")
			nw.checksumButtonFunc()
		}
	})
	nw.filenameLabel = widget.NewLabel("")

	return nw
}

func (nw *nodeWidget) SetLabel(name string) {
	nw.filenameLabel.SetText(name)
}

func (nw *nodeWidget) SetButtonFunc(f func()) {
	nw.checksumButtonFuncLock.Lock()
	defer nw.checksumButtonFuncLock.Unlock()
	nw.checksumButtonFunc = f
}

func (nw *nodeWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewHBox(nw.filenameLabel, nw.checksumButton))
}
