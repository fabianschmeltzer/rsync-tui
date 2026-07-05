package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type hitTarget struct {
	kind       string
	index      int
	minX, maxX int
	minY, maxY int
}

type targetSpec struct {
	kind   string
	index  int
	needle string
	tall   bool
}

func (m Model) handleMouseMotion(message tea.MouseMotionMsg) (tea.Model, tea.Cmd) {
	mouse := message.Mouse()
	target, ok := m.hitTest(mouse.X, mouse.Y)
	if !ok {
		m.hoverKind = ""
		m.hoverIndex = -1
		return m, nil
	}
	m.hoverKind = target.kind
	m.hoverIndex = target.index
	return m, nil
}

func (m Model) handleMouseClick(message tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	mouse := message.Mouse()
	if mouse.Button != tea.MouseLeft {
		return m, nil
	}
	target, ok := m.hitTest(mouse.X, mouse.Y)
	if !ok {
		return m, nil
	}
	m.hoverKind = target.kind
	m.hoverIndex = target.index
	switch target.kind {
	case "home":
		m.screen = screenHome
		if target.index < 0 {
			return m, nil
		}
		m.cursor = target.index
		return m.handleHome("enter")
	case "profile":
		m.cursor = target.index
		return m.handleProfiles("enter")
	case "setting":
		m.settingsCursor = target.index
		return m.handleSettings("enter")
	case "history":
		m.historyCursor = target.index
		return m.handleHistory("enter")
	case "browser":
		m.browserCursor = target.index
		return m.handleBrowser("enter")
	case "wizard-choice":
		m.profileChoice = target.index
		return m.handleWizard("enter")
	case "wizard-mode":
		m.modeCursor = target.index
		return m.handleWizard("enter")
	case "wizard-advanced":
		m.advancedCursor = target.index
		return m.handleWizard("space")
	case "cancel":
		if m.cancel != nil {
			m.cancel()
			m.status = m.translator.T("status.cancelling")
		}
		return m, nil
	}
	return m, nil
}

func (m Model) hitTest(x, y int) (hitTarget, bool) {
	targets := m.hitTargets()
	for index := len(targets) - 1; index >= 0; index-- {
		target := targets[index]
		if x >= target.minX && x <= target.maxX && y >= target.minY && y <= target.maxY {
			return target, true
		}
	}
	return hitTarget{}, false
}

func (m Model) hitTargets() []hitTarget {
	content := ansi.Strip(m.renderApplication())
	lines := strings.Split(content, "\n")
	var specs []targetSpec

	if m.navigationVisible() {
		specs = append(specs, targetSpec{"home", -1, m.translator.T("page.dashboard"), false})
		for index, label := range m.homeItems() {
			specs = append(specs, targetSpec{"home", index, label, true})
		}
	}
	switch m.screen {
	case screenHome:
		for index, label := range m.homeItems() {
			specs = append(specs, targetSpec{"home", index, label, true})
		}
	case screenProfiles:
		for index, profile := range m.profiles {
			specs = append(specs, targetSpec{"profile", index, profile.Name, true})
		}
	case screenSettings:
		for index, label := range m.settingLabels() {
			specs = append(specs, targetSpec{"setting", index, label + ":", false})
		}
	case screenHistory:
		if !m.historyDetail {
			for index, entry := range m.history {
				needle := formatHistoryTime(entry.StartedAt) + "  " + m.historyEntryName(entry)
				specs = append(specs, targetSpec{"history", index, needle, true})
			}
		}
	case screenBrowser:
		for index, entry := range m.browserEntries {
			specs = append(specs, targetSpec{"browser", index, entry.Name + "/", true})
		}
	case screenRunning:
		specs = append(specs, targetSpec{"cancel", 0, m.translator.T("action.cancel"), false})
	case screenWizard:
		switch m.wizardStage {
		case wizardChooseStorage:
			specs = append(specs,
				targetSpec{"wizard-choice", 0, m.translator.T("wizard.storage.one_time"), false},
				targetSpec{"wizard-choice", 1, m.translator.T("wizard.storage.profile"), false})
		case wizardMode:
			for index, mode := range wizardModes(m.saveProfile) {
				specs = append(specs, targetSpec{"wizard-mode", index, m.translator.T("wizard.mode." + string(mode)), false})
			}
		case wizardAdvanced:
			for index, option := range wizardAdvancedOptions() {
				specs = append(specs, targetSpec{"wizard-advanced", index, option, false})
			}
		}
	}

	var targets []hitTarget
	for _, spec := range specs {
		if strings.TrimSpace(spec.needle) == "" {
			continue
		}
		for y, line := range lines {
			searchFrom := 0
			for {
				position := strings.Index(line[searchFrom:], spec.needle)
				if position < 0 {
					break
				}
				position += searchFrom
				x := lipgloss.Width(line[:position])
				minY, maxY := y, y
				tall := spec.tall
				if spec.kind == "home" {
					shellWidth := min(144, m.width)
					tall = m.screen == screenHome &&
						responsiveLayout(shellWidth) != layoutCompact &&
						m.height >= 28 &&
						x >= 24
				}
				if tall {
					minY = max(0, y-1)
					maxY = y + 2
				}
				targets = append(targets, hitTarget{
					kind:  spec.kind,
					index: spec.index,
					minX:  max(0, x-2),
					maxX:  x + max(12, lipgloss.Width(spec.needle)+24),
					minY:  minY,
					maxY:  maxY,
				})
				searchFrom = position + len(spec.needle)
			}
		}
	}
	return targets
}

func (m Model) settingLabels() []string {
	return []string{
		m.translator.T("settings.language"),
		m.translator.T("settings.theme"),
		m.translator.T("settings.accent"),
		m.translator.T("settings.density"),
		m.translator.T("settings.icons"),
		m.translator.T("settings.motion"),
		m.translator.T("settings.auto_update"),
		m.translator.T("settings.update_channel"),
		m.translator.T("settings.check_hours"),
	}
}
