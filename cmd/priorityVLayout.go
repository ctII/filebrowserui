package cmd

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
)

// priorityVLayout layouts all objects as min sized, but with the first CanvasObject taking the
// rest of the height. if len([]fyne.CanvasObjects) < 2 we act the same as a layout.StackLayout
type priorityVLayout struct{}

var _ fyne.Layout = &priorityVLayout{}

func (psl *priorityVLayout) Layout(objs []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objs) < 2 {
		layout.NewStackLayout().Layout(objs, containerSize)
		return
	}

	takenHeight := float32(0)
	for i := len(objs) - 1; i != 0; i-- {
		objMinSize := objs[i].MinSize()
		takenHeight += objMinSize.Height

		objs[i].Resize(fyne.NewSize(containerSize.Width, objMinSize.Height))
		objs[i].Move(fyne.NewPos(0, containerSize.Height-takenHeight))
	}

	objs[0].Resize(containerSize.SubtractWidthHeight(0, takenHeight))
}

func (psl *priorityVLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return layout.NewStackLayout().MinSize(objects)
	}

	minWidth, minHeight := float32(0), float32(0)
	for _, obj := range objects {
		objMinSize := obj.MinSize()

		minWidth += objMinSize.Width
		minHeight += objMinSize.Height
	}

	return fyne.NewSize(minWidth, minHeight)
}
