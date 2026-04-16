// Package gui implements the native cross-platform desktop interface for tglys
// using the Fyne toolkit (fyne.io/fyne/v2).
//
// The application runs as a system-tray background job. The tray icon reflects
// the current traffic-light state (grey = unknown, red/yellow/green). Clicking
// "Settings" in the tray menu opens a window where the user can inspect the
// current state and edit all configuration values.
package gui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/VidarTessem/tglys/checker"
	"github.com/VidarTessem/tglys/config"
)

// colorCircle is a fixed-size custom widget that renders a filled circle.
type colorCircle struct {
	widget.BaseWidget
	circle *canvas.Circle
}

func newColorCircle(c color.Color) *colorCircle {
	cc := &colorCircle{circle: canvas.NewCircle(c)}
	cc.ExtendBaseWidget(cc)
	return cc
}

func (cc *colorCircle) MinSize() fyne.Size { return fyne.NewSize(80, 80) }

func (cc *colorCircle) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(cc.circle)
}

func (cc *colorCircle) setColor(c color.Color) {
	cc.circle.FillColor = c
	cc.circle.Refresh()
}

// Run starts the Fyne application. It must be called on the main goroutine and
// blocks until the window is closed (which also terminates the process).
//
// ch is the running Checker instance.
// getCfg returns the current live Config.
// setCfg saves a new Config (persisted to .env) and returns any error.
func Run(ch *checker.Checker, getCfg func() config.Config, setCfg func(config.Config) error) {
	a := app.NewWithID("no.tg.tglys")
	a.SetIcon(circleIcon(colorUnknown))

	win := a.NewWindow("tglys — TG Traffic Light")
	win.SetFixedSize(true)
	win.Resize(fyne.NewSize(420, 520))
	win.SetCloseIntercept(func() {
		win.Hide()
	})

	// ── Status card ────────────────────────────────────────────────────────
	bulb := newColorCircle(colorUnknown)

	stateLabel := widget.NewLabel("unknown")
	stateLabel.TextStyle = fyne.TextStyle{Bold: true}
	stateLabel.Alignment = fyne.TextAlignCenter

	errorLabel := widget.NewLabel("")
	errorLabel.Wrapping = fyne.TextWrapWord

	statusCard := container.NewVBox(
		container.NewCenter(bulb),
		container.NewCenter(stateLabel),
		errorLabel,
	)

	// ── Config form ─────────────────────────────────────────────────────────
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("https://example.com/api/trafikklys")

	pollEntry := widget.NewEntry()
	pollEntry.SetPlaceHolder("30")

	redEntry := widget.NewEntry()
	redEntry.SetPlaceHolder("0")

	yellowEntry := widget.NewEntry()
	yellowEntry.SetPlaceHolder("50")

	greenEntry := widget.NewEntry()
	greenEntry.SetPlaceHolder("100")

	statusMsg := widget.NewLabel("")
	statusMsg.Alignment = fyne.TextAlignCenter

	saveBtn := widget.NewButton("Save & Apply", func() {
		cfg := getCfg()

		cfg.URL = urlEntry.Text

		if n, err := strconv.Atoi(pollEntry.Text); err == nil && n > 0 {
			cfg.PollInterval = n
		}
		if n, err := strconv.Atoi(redEntry.Text); err == nil {
			cfg.RedVolume = clamp(n, 0, 100)
		}
		if n, err := strconv.Atoi(yellowEntry.Text); err == nil {
			cfg.YellowVolume = clamp(n, 0, 100)
		}
		if n, err := strconv.Atoi(greenEntry.Text); err == nil {
			cfg.GreenVolume = clamp(n, 0, 100)
		}

		if err := setCfg(cfg); err != nil {
			statusMsg.SetText("⚠ " + err.Error())
			log.Printf("[gui] save error: %v", err)
			return
		}
		statusMsg.SetText("✓ Saved!")
		go func() {
			time.Sleep(3 * time.Second)
			statusMsg.SetText("")
		}()
	})

	form := widget.NewForm(
		widget.NewFormItem("API URL", urlEntry),
		widget.NewFormItem("Poll interval (s)", pollEntry),
		widget.NewFormItem("🔴 Red volume %", redEntry),
		widget.NewFormItem("🟡 Yellow volume %", yellowEntry),
		widget.NewFormItem("🟢 Green volume %", greenEntry),
	)

	configCard := container.NewVBox(
		form,
		container.NewCenter(saveBtn),
		container.NewCenter(statusMsg),
	)

	// ── Layout ──────────────────────────────────────────────────────────────
	win.SetContent(container.NewVBox(
		widget.NewSeparator(),
		statusCard,
		widget.NewSeparator(),
		widget.NewLabel("Configuration"),
		configCard,
		layout.NewSpacer(),
	))

	// ── System tray (desktop platforms only) ─────────────────────────────────
	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayIcon(circleIcon(colorUnknown))
		desk.SetSystemTrayMenu(fyne.NewMenu("tglys",
			fyne.NewMenuItem("Open Settings", func() {
				win.Show()
				win.RequestFocus()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				a.Quit()
			}),
		))
	}

	// ── Populate form fields once on start ──────────────────────────────────
	populateForm := func() {
		cfg := getCfg()
		urlEntry.SetText(cfg.URL)
		pollEntry.SetText(strconv.Itoa(cfg.PollInterval))
		redEntry.SetText(strconv.Itoa(cfg.RedVolume))
		yellowEntry.SetText(strconv.Itoa(cfg.YellowVolume))
		greenEntry.SetText(strconv.Itoa(cfg.GreenVolume))
	}
	populateForm()

	// ── Background refresh loop ──────────────────────────────────────────────
	go func() {
		var lastState checker.State = "init"
		for {
			time.Sleep(time.Second)

			state := ch.Current()
			lastErr := ch.LastError()

			c := stateColor(state)
			bulb.setColor(c)

			label := string(state)
			if label == "" {
				label = "unknown"
			}
			stateLabel.SetText(label)

			if lastErr != nil {
				errorLabel.SetText("⚠ " + lastErr.Error())
			} else {
				errorLabel.SetText("")
			}

			// Update tray icon when state changes.
			if state != lastState {
				icon := circleIcon(c)
				a.SetIcon(icon)
				if desk, ok := a.(desktop.App); ok {
					desk.SetSystemTrayIcon(icon)
				}
				lastState = state
			}
		}
	}()

	win.ShowAndRun()
}

// stateColor maps a traffic-light state to a display colour.
func stateColor(s checker.State) color.Color {
	switch s {
	case checker.StateRed:
		return colorRed
	case checker.StateYellow:
		return colorYellow
	case checker.StateGreen:
		return colorGreen
	default:
		return colorUnknown
	}
}

// Named colours for the traffic light.
var (
	colorRed     = color.RGBA{R: 230, G: 57, B: 70, A: 255}
	colorYellow  = color.RGBA{R: 244, G: 162, B: 97, A: 255}
	colorGreen   = color.RGBA{R: 42, G: 157, B: 143, A: 255}
	colorUnknown = color.RGBA{R: 120, G: 120, B: 120, A: 255}
)

// circleIcon returns a 64×64 PNG fyne.Resource containing a filled circle of
// the given colour, suitable for use as a system tray or app icon.
func circleIcon(c color.Color) fyne.Resource {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 2

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if math.Sqrt(dx*dx+dy*dy) <= r {
				img.Set(x, y, c)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Printf("[gui] icon encode error: %v", err)
	}
	return fyne.NewStaticResource(fmt.Sprintf("icon-%v.png", c), buf.Bytes())
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

