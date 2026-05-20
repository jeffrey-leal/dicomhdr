package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// treeRow is the canvas object used for each row in the tag tree.
// It wraps a canvas.Text with rowLayout, supports right-click context menus
// via fyne.SecondaryTappable, and shows a DICOM standard tooltip on hover.
type treeRow struct {
	widget.BaseWidget
	ct          *canvas.Text
	nodeID      string
	tooltipText string        // pre-computed in updateNode; empty → no tooltip
	cv          fyne.Canvas   // used to anchor the popup
	hoverPos    fyne.Position // updated by MouseMoved; read by the timer callback
	hoverTimer  *time.Timer
	showPending bool // cleared by MouseOut to cancel an in-flight timer
	tooltipPop  *widget.PopUp
	onMenu      func(id string, pos fyne.Position)
}

func newTreeRow(cv fyne.Canvas, onMenu func(id string, pos fyne.Position)) *treeRow {
	tr := &treeRow{
		ct:     canvas.NewText("", theme.Color(theme.ColorNameForeground)),
		cv:     cv,
		onMenu: onMenu,
	}
	tr.ExtendBaseWidget(tr)
	return tr
}

func (tr *treeRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.New(rowLayout{}, tr.ct))
}

// ── Secondary tap (right-click) ───────────────────────────────────────────────

func (tr *treeRow) TappedSecondary(e *fyne.PointEvent) {
	if tr.onMenu != nil && tr.nodeID != "" {
		tr.onMenu(tr.nodeID, e.AbsolutePosition)
	}
}

// ── Hover / tooltip ───────────────────────────────────────────────────────────

func (tr *treeRow) MouseIn(e *desktop.MouseEvent) {
	tr.hideTooltip()
	if tr.tooltipText == "" {
		return
	}
	tr.showPending = true
	tr.hoverPos = e.AbsolutePosition
	tr.hoverTimer = time.AfterFunc(600*time.Millisecond, func() {
		fyne.Do(func() {
			if tr.showPending {
				tr.showTooltip()
			}
		})
	})
}

func (tr *treeRow) MouseMoved(e *desktop.MouseEvent) {
	// Track position so the tooltip appears where the cursor settled, not
	// where it first entered the row.
	tr.hoverPos = e.AbsolutePosition
}

func (tr *treeRow) MouseOut() {
	tr.hideTooltip()
}

func (tr *treeRow) showTooltip() {
	if tr.cv == nil || tr.tooltipText == "" {
		return
	}
	lbl := widget.NewLabel(tr.tooltipText)
	lbl.TextStyle = fyne.TextStyle{Monospace: true}
	tr.tooltipPop = widget.NewPopUp(container.NewPadded(lbl), tr.cv)
	tr.tooltipPop.ShowAtPosition(fyne.NewPos(tr.hoverPos.X+12, tr.hoverPos.Y+16))
}

func (tr *treeRow) hideTooltip() {
	tr.showPending = false
	if tr.hoverTimer != nil {
		tr.hoverTimer.Stop()
		tr.hoverTimer = nil
	}
	if tr.tooltipPop != nil {
		tr.tooltipPop.Hide()
		tr.tooltipPop = nil
	}
}
