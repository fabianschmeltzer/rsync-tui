package tui

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	"github.com/fabianschmeltzer/rsync-tui/internal/job"
	rsyncengine "github.com/fabianschmeltzer/rsync-tui/internal/rsync"
)

func TestMaterialPalettesMeetTextContrast(t *testing.T) {
	themes := []string{"material-dark", "material-light", "midnight", "high-contrast"}
	accents := []string{"indigo", "blue", "teal", "green", "amber", "rose", "violet"}
	for _, theme := range themes {
		for _, accent := range accents {
			t.Run(theme+"/"+accent, func(t *testing.T) {
				roles := materialRoles(theme, accent)
				assertContrast(t, roles.OnSurface, roles.Background, 4.5)
				assertContrast(t, roles.OnPrimaryContainer, roles.PrimaryContainer, 4.5)
			})
		}
	}
}

func TestResponsiveRenderStaysInsideTerminal(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {46, 18}, {60, 24}, {90, 30}, {120, 36}, {160, 42}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			model := newDesignTestModel(t)
			model.width, model.height = size[0], size[1]
			assertRenderFits(t, model.render(), size[0], size[1])
		})
	}
}

func TestAllScreensRenderInMaterialThemes(t *testing.T) {
	for _, theme := range []string{"material-dark", "material-light"} {
		for _, language := range []string{"en", "de"} {
			t.Run(theme+"/"+language, func(t *testing.T) {
				model := newDesignTestModel(t)
				model.settings.Theme = theme
				model.settings.Language = language
				model.design = newDesignSystem(model.settings, false)
				model.translator.Language = language
				model.width, model.height = 110, 36
				model.draft.Source.Path = "/source"
				model.draft.Destination.Path = "/destination"
				model.selected = model.draft
				model.profiles = []domain.Profile{model.draft}
				model.browserEntries = nil
				model.history = []rsyncengine.Result{{ProfileName: "Test", Mode: domain.ModeCopy}}
				model.lastOutcome = job.Outcome{Result: rsyncengine.Result{ProfileName: "Test", Mode: domain.ModeCopy}}
				for _, current := range []screen{
					screenHome,
					screenWizard,
					screenProfiles,
					screenRunning,
					screenResult,
					screenSettings,
					screenHistory,
					screenBrowser,
				} {
					model.screen = current
					rendered := model.render()
					if strings.TrimSpace(ansi.Strip(rendered)) == "" {
						t.Fatalf("screen %d rendered empty", current)
					}
					assertRenderFits(t, rendered, model.width, model.height)
				}
			})
		}
	}
}

func TestMouseTargetsHoverAndActivateDashboardCard(t *testing.T) {
	model := newDesignTestModel(t)
	model.width, model.height = 100, 30
	var target hitTarget
	found := false
	for _, candidate := range model.hitTargets() {
		if candidate.kind == "home" && candidate.index == 0 {
			target, found = candidate, true
			break
		}
	}
	if !found {
		t.Fatal("new transfer card has no mouse target")
	}
	x := (target.minX + target.maxX) / 2
	y := (target.minY + target.maxY) / 2
	updated, _ := model.handleMouseMotion(tea.MouseMotionMsg(tea.Mouse{X: x, Y: y}))
	model = updated.(Model)
	if !model.isHovered("home", 0) {
		t.Fatal("mouse motion did not set hover state")
	}
	updated, _ = model.handleMouseClick(tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft}))
	model = updated.(Model)
	if model.screen != screenWizard {
		t.Fatalf("mouse click opened screen %d, want wizard", model.screen)
	}
}

func TestIconAndMotionModesAreDistinct(t *testing.T) {
	settings := config.DefaultSettings()
	unicode := newDesignSystem(settings, false)
	settings.Icons = "nerd-font"
	nerd := newDesignSystem(settings, false)
	if unicode.Icons.New == nerd.Icons.New {
		t.Fatal("Nerd Font mode did not change icons")
	}

	model := newDesignTestModel(t)
	model.width = 80
	model.settings.Motion = "none"
	static := model.renderProgressIndicator(30)
	model.settings.Motion = "expressive"
	expressive := model.renderProgressIndicator(30)
	if ansi.Strip(static) == ansi.Strip(expressive) {
		t.Fatal("motion modes render identical progress indicators")
	}
	if newActivitySpinner(model.design, "expressive").Spinner.FPS >= newActivitySpinner(model.design, "subtle").Spinner.FPS {
		t.Fatal("expressive motion did not speed up the activity spinner")
	}
}

func TestMouseWheelScrollsRunningLogAndCancelIsClickable(t *testing.T) {
	model := newDesignTestModel(t)
	model.width, model.height = 90, 24
	model.screen = screenRunning
	for index := 0; index < 30; index++ {
		model.logLines = append(model.logLines, fmt.Sprintf("line %02d", index))
	}
	updated, _ := model.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	model = updated.(Model)
	if model.logOffset == 0 {
		t.Fatal("mouse wheel did not scroll the running log")
	}

	cancelled := false
	model.cancel = func() { cancelled = true }
	var cancelTarget hitTarget
	found := false
	for _, candidate := range model.hitTargets() {
		if candidate.kind == "cancel" {
			cancelTarget, found = candidate, true
			break
		}
	}
	if !found {
		t.Fatal("running screen has no cancel hit target")
	}
	updated, _ = model.handleMouseClick(tea.MouseClickMsg(tea.Mouse{
		X:      (cancelTarget.minX + cancelTarget.maxX) / 2,
		Y:      (cancelTarget.minY + cancelTarget.maxY) / 2,
		Button: tea.MouseLeft,
	}))
	model = updated.(Model)
	if !cancelled || model.status == "" {
		t.Fatal("clicking cancel did not cancel the running transfer")
	}
	if !strings.Contains(ansi.Strip(model.render()), model.status) {
		t.Fatal("cancellation status is not visible as a snackbar")
	}
}

func newDesignTestModel(t *testing.T) Model {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	model := New(store, config.DefaultSettings(), "0.1.3")
	model.draft = domain.NewProfile("Test")
	return model
}

func assertContrast(t *testing.T, foreground, background string, minimum float64) {
	t.Helper()
	ratio := contrastRatio(foreground, background)
	if ratio < minimum {
		t.Fatalf("contrast %s on %s is %.2f, want at least %.2f", foreground, background, ratio, minimum)
	}
}

func assertRenderFits(t *testing.T, rendered string, width, height int) {
	t.Helper()
	lines := strings.Split(ansi.Strip(rendered), "\n")
	if len(lines) > height {
		t.Fatalf("rendered %d lines into terminal height %d", len(lines), height)
	}
	for index, line := range lines {
		if renderedWidth := lipgloss.Width(line); renderedWidth > width {
			t.Fatalf("line %d is %d cells wide in terminal width %d", index, renderedWidth, width)
		}
	}
}

func contrastRatio(first, second string) float64 {
	a, b := relativeLuminance(first), relativeLuminance(second)
	if a < b {
		a, b = b, a
	}
	return (a + 0.05) / (b + 0.05)
}

func relativeLuminance(value string) float64 {
	value = strings.TrimPrefix(value, "#")
	channel := func(offset int) float64 {
		parsed, _ := strconv.ParseUint(value[offset:offset+2], 16, 8)
		component := float64(parsed) / 255
		if component <= 0.04045 {
			return component / 12.92
		}
		return math.Pow((component+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(0) + 0.7152*channel(2) + 0.0722*channel(4)
}
