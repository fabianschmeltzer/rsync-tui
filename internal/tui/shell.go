package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	rsyncengine "github.com/fabianschmeltzer/rsync-tui/internal/rsync"
)

func loadDashboard(store *config.Store) tea.Cmd {
	return func() tea.Msg {
		result := dashboardLoadedMsg{}
		profiles, err := store.ListProfiles()
		if err == nil {
			result.profiles = len(profiles)
			for _, profile := range profiles {
				if profile.Mode == "snapshot" {
					result.snapshots++
				}
				if profile.Schedule.Enabled {
					result.schedules++
				}
			}
		}
		history, err := rsyncengine.LoadHistory(store.Paths.StateDir, 3)
		if err == nil {
			result.history = history.Entries
		}
		return result
	}
}

func (m Model) renderApplication() string {
	width, height := m.width, m.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	if width < 46 || height < 18 {
		return m.renderTooSmall(width, height)
	}

	shellWidth := min(144, width)
	mode := responsiveLayout(shellWidth)
	showNavigation := mode != layoutCompact && m.navigationVisible()
	navigationWidth := 24
	contentWidth := shellWidth - 4
	if showNavigation {
		contentWidth -= navigationWidth + m.design.Gap
	}
	body := m.renderScreenContent(max(32, contentWidth), mode)

	var content string
	if showNavigation {
		navigation := m.renderNavigationRail(navigationWidth)
		main := m.design.Panel.Width(max(28, contentWidth-2*m.design.Padding)).Render(body)
		content = lipgloss.JoinHorizontal(lipgloss.Top, navigation, strings.Repeat(" ", m.design.Gap), main)
	} else {
		content = m.design.Panel.Width(max(30, contentWidth-2*m.design.Padding)).Render(body)
	}

	appBar := m.renderAppBar(shellWidth - 4)
	footer := m.design.shortcutBar(m.screenHelp(), shellWidth-4)
	parts := []string{appBar, "", content, "", footer}
	shell := strings.Join(parts, "\n")
	left := max(0, (width-shellWidth)/2)
	frame := lipgloss.NewStyle().MarginLeft(left).Render(shell)
	snackbar := ""
	if m.status != "" && (m.screen == screenHome || m.screen == screenSettings || m.screen == screenRunning) {
		snackbar = m.design.snackbar(m.status, shellWidth-4)
	}
	contentHeight := height
	if snackbar != "" {
		contentHeight--
	}
	rendered := m.design.Background.Width(width).Height(contentHeight).Render(fitTerminal(frame, width, contentHeight))
	if snackbar == "" {
		return rendered
	}
	bottom := ansi.Truncate(strings.Repeat(" ", left)+snackbar, width, "")
	return rendered + "\n" + m.design.Background.Width(width).Render(bottom)
}

func (m Model) renderScreenContent(width int, mode layoutMode) string {
	switch m.screen {
	case screenHome:
		return m.renderDashboard(width, mode)
	case screenWizard:
		return m.renderWizard()
	case screenProfiles:
		return m.renderProfiles()
	case screenRunning:
		return m.renderRunning()
	case screenResult:
		return m.renderResult()
	case screenInfo:
		return m.design.CardHigh.Width(max(24, width-2*m.design.Padding)).Render(m.status)
	case screenSettings:
		return m.renderSettings()
	case screenHistory:
		return m.renderHistory()
	case screenBrowser:
		return m.renderBrowser()
	default:
		return ""
	}
}

func (m Model) navigationVisible() bool {
	switch m.screen {
	case screenHome, screenProfiles, screenInfo, screenSettings, screenHistory:
		return true
	default:
		return false
	}
}

func (m Model) renderAppBar(width int) string {
	left := m.design.Icons.App + "  rsync-tui  " + m.design.Subtitle.Render("· "+m.pageTitle())
	language := strings.ToUpper(m.translator.Language)
	right := m.design.chip("v"+m.version, false) + " " +
		m.design.chip(language, false) + " " +
		m.design.chip(m.translator.T("settings.theme."+m.settings.Theme), true)
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right)-4)
	return m.design.AppBar.Width(max(20, width-4)).Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) pageTitle() string {
	switch m.screen {
	case screenHome:
		return m.translator.T("page.dashboard")
	case screenWizard:
		return m.translator.T("menu.new")
	case screenProfiles:
		return m.translator.T("menu.profiles")
	case screenRunning:
		return m.translator.T("page.running")
	case screenResult:
		return m.translator.T("page.result")
	case screenSettings:
		return m.translator.T("menu.settings")
	case screenHistory:
		return m.translator.T("menu.history")
	case screenBrowser:
		return m.translator.T("page.browser")
	default:
		return m.translator.T("app.subtitle")
	}
}

func (m Model) renderNavigationRail(width int) string {
	items := []struct {
		icon  string
		label string
		index int
	}{
		{m.design.Icons.Home, m.translator.T("page.dashboard"), -1},
		{m.design.Icons.New, m.translator.T("menu.new"), 0},
		{m.design.Icons.Profiles, m.translator.T("menu.profiles"), 1},
		{m.design.Icons.Snapshots, m.translator.T("menu.snapshots"), 2},
		{m.design.Icons.Schedules, m.translator.T("menu.schedules"), 3},
		{m.design.Icons.History, m.translator.T("menu.history"), 4},
		{m.design.Icons.Settings, m.translator.T("menu.settings"), 5},
	}
	active := m.activeNavigationIndex()
	lines := []string{m.design.Title.Render(m.design.Icons.App + "  rsync-tui"), ""}
	for _, item := range items {
		label := truncateDisplay(fmt.Sprintf("%s  %s", item.icon, item.label), width-4)
		style := m.design.Item
		if item.index == active || (item.index == -1 && active == -1) {
			style = m.design.Selected
		}
		lines = append(lines, style.Width(max(12, width-4)).Render(label))
	}
	lines = append(lines, "", m.design.Subtitle.Render(m.design.Icons.Quit+"  "+m.translator.T("menu.quit")))
	return m.design.CardHigh.Width(max(14, width-2*m.design.Padding)).Render(strings.Join(lines, "\n"))
}

func (m Model) activeNavigationIndex() int {
	switch m.screen {
	case screenProfiles:
		return 1
	case screenSettings:
		return 5
	case screenHistory:
		return 4
	case screenInfo:
		switch m.cursor {
		case 2, 3:
			return m.cursor
		}
	}
	return -1
}

func (m Model) renderDashboard(width int, mode layoutMode) string {
	if mode == layoutCompact || m.height < 28 {
		return m.renderCompactDashboard(width)
	}
	cardWidth := max(24, width-2*m.design.Padding-2)
	newCard := m.design.card(
		m.design.Icons.New+"  "+m.translator.T("menu.new"),
		m.translator.T("dashboard.new.description"),
		cardWidth,
		m.cursor == 0,
		m.isHovered("home", 0),
		true,
	)
	profiles := m.design.card(
		m.design.Icons.Profiles+"  "+m.translator.T("menu.profiles"),
		m.translator.T("dashboard.count", m.dashboard.profiles),
		cardWidth,
		m.cursor == 1,
		m.isHovered("home", 1),
		false,
	)
	snapshots := m.design.card(
		m.design.Icons.Snapshots+"  "+m.translator.T("menu.snapshots"),
		m.translator.T("dashboard.count", m.dashboard.snapshots),
		cardWidth,
		m.cursor == 2,
		m.isHovered("home", 2),
		false,
	)
	schedules := m.design.card(
		m.design.Icons.Schedules+"  "+m.translator.T("menu.schedules"),
		m.translator.T("dashboard.count", m.dashboard.schedules),
		cardWidth,
		m.cursor == 3,
		m.isHovered("home", 3),
		false,
	)
	history := m.design.card(
		m.design.Icons.History+"  "+m.translator.T("menu.history"),
		m.dashboardHistorySummary(),
		cardWidth,
		m.cursor == 4,
		m.isHovered("home", 4),
		false,
	)
	settings := m.design.card(
		m.design.Icons.Settings+"  "+m.translator.T("menu.settings"),
		m.translator.T("dashboard.appearance", m.translator.T("settings.theme."+m.settings.Theme), m.translator.T("settings.accent."+m.settings.Accent)),
		cardWidth,
		m.cursor == 5,
		m.isHovered("home", 5),
		false,
	)

	if mode == layoutExpanded {
		columnWidth := max(24, (width-m.design.Gap)/2)
		left := strings.Join([]string{
			m.design.Headline.Render(m.translator.T("dashboard.welcome")),
			m.design.Subtitle.Render(m.translator.T("dashboard.subtitle")),
			"",
			resizeCard(newCard, columnWidth),
			"",
			resizeCard(history, columnWidth),
		}, "\n")
		right := strings.Join([]string{
			resizeCard(profiles, columnWidth),
			"",
			resizeCard(snapshots, columnWidth),
			"",
			resizeCard(schedules, columnWidth),
			"",
			resizeCard(settings, columnWidth),
		}, "\n")
		return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", m.design.Gap), right)
	}

	return strings.Join([]string{
		m.design.Headline.Render(m.translator.T("dashboard.welcome")),
		m.design.Subtitle.Render(m.translator.T("dashboard.subtitle")),
		"",
		newCard,
		"",
		profiles,
		"",
		snapshots,
		"",
		schedules,
		"",
		history,
		"",
		settings,
	}, "\n")
}

func (m Model) renderCompactDashboard(width int) string {
	items := []struct {
		icon  string
		label string
		meta  string
	}{
		{m.design.Icons.New, m.translator.T("menu.new"), m.translator.T("dashboard.new.description")},
		{m.design.Icons.Profiles, m.translator.T("menu.profiles"), m.translator.T("dashboard.count", m.dashboard.profiles)},
		{m.design.Icons.Snapshots, m.translator.T("menu.snapshots"), m.translator.T("dashboard.count", m.dashboard.snapshots)},
		{m.design.Icons.Schedules, m.translator.T("menu.schedules"), m.translator.T("dashboard.count", m.dashboard.schedules)},
		{m.design.Icons.History, m.translator.T("menu.history"), m.translator.T("dashboard.recent")},
		{m.design.Icons.Settings, m.translator.T("menu.settings"), m.translator.T("settings.theme." + m.settings.Theme)},
		{m.design.Icons.Quit, m.translator.T("menu.quit"), ""},
	}
	lines := make([]string, 0, len(items))
	for index, item := range items {
		line := fmt.Sprintf("%s  %-22s %s", item.icon, item.label, item.meta)
		line = truncateDisplay(line, max(24, width-4))
		style := m.design.Item
		if m.isHovered("home", index) {
			style = m.design.Hover
		}
		if m.cursor == index {
			style = m.design.Selected
		}
		lines = append(lines, style.Render(line))
	}
	return m.design.Headline.Render(m.translator.T("dashboard.welcome")) + "\n" +
		m.design.Subtitle.Render(m.translator.T("dashboard.subtitle")) + "\n\n" +
		m.design.CardHigh.Render(strings.Join(lines, "\n"))
}

func resizeCard(card string, width int) string {
	return lipgloss.NewStyle().Width(max(20, width)).Render(card)
}

func (m Model) dashboardHistorySummary() string {
	if len(m.dashboard.history) == 0 {
		return m.translator.T("history.empty")
	}
	lines := make([]string, 0, len(m.dashboard.history))
	for _, entry := range m.dashboard.history {
		icon := m.design.Icons.Success
		if entry.ExitCode != 0 {
			icon = m.design.Icons.Error
		}
		lines = append(lines, fmt.Sprintf("%s %s · %s", icon, m.historyEntryName(entry), formatHistoryTime(entry.StartedAt)))
	}
	return strings.Join(lines, "\n")
}

func (m Model) screenHelp() string {
	switch m.screen {
	case screenHome:
		return m.translator.T("help.navigation")
	case screenWizard:
		return m.translator.T("wizard.storage.help")
	case screenHistory:
		if m.historyDetail {
			return m.translator.T("history.help.detail")
		}
		return m.translator.T("history.help.list")
	case screenRunning:
		return "Ctrl+C — " + m.translator.T("action.cancel")
	default:
		return m.translator.T("help.back")
	}
}

func (m Model) renderTooSmall(width, height int) string {
	message := fmt.Sprintf("%s\n\n%s\n%s",
		m.design.Icons.Info+"  "+m.translator.T("terminal.small.title"),
		m.translator.T("terminal.small.message"),
		m.translator.T("terminal.small.size", width, height))
	card := m.design.CardPrimary.Render(message)
	return m.design.Background.Width(max(1, width)).Height(max(1, height)).Align(lipgloss.Center).Render(card)
}

func fitTerminal(content string, width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for index, line := range lines {
		lines[index] = ansi.Truncate(line, width, "")
	}
	return strings.Join(lines, "\n")
}
