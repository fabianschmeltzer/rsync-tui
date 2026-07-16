package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/fabianschmeltzer/rsync-tui/internal/browser"
	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	"github.com/fabianschmeltzer/rsync-tui/internal/i18n"
	"github.com/fabianschmeltzer/rsync-tui/internal/job"
	rsyncengine "github.com/fabianschmeltzer/rsync-tui/internal/rsync"
	"github.com/fabianschmeltzer/rsync-tui/internal/sshclient"
)

type screen int

const (
	screenHome screen = iota
	screenWizard
	screenProfiles
	screenRunning
	screenResult
	screenInfo
	screenSettings
	screenHistory
	screenBrowser
)

type wizardStage int

const (
	wizardChooseStorage wizardStage = iota
	wizardName
	wizardSource
	wizardDestination
	wizardMode
	wizardAdvanced
	wizardExpert
	wizardReview
)

type runEventMsg rsyncengine.Event

type runFinishedMsg struct {
	outcome job.Outcome
	err     error
}

type sshReadyMsg struct {
	err error
}

type sudoReadyMsg struct {
	err error
}

type browserLoadedMsg struct {
	entries []browser.Entry
	err     error
}

type dashboardLoadedMsg struct {
	profiles  int
	snapshots int
	schedules int
	history   []rsyncengine.Result
}

// Model contains the terminal user-interface state.
type Model struct {
	store            *config.Store
	settings         config.Settings
	design           designSystem
	translator       i18n.Translator
	version          string
	width            int
	height           int
	screen           screen
	cursor           int
	settingsCursor   int
	status           string
	input            textinput.Model
	wizardStage      wizardStage
	saveProfile      bool
	profileChoice    int
	wizardName       string
	draft            domain.Profile
	modeCursor       int
	advancedCursor   int
	expertOptions    []string
	dryRun           bool
	confirm          int
	profiles         []domain.Profile
	selected         domain.Profile
	spinner          spinner.Model
	runEvents        chan tea.Msg
	cancel           context.CancelFunc
	logLines         []string
	logOffset        int
	lastOutcome      job.Outcome
	lastErr          error
	pendingProfile   domain.Profile
	pendingDryRun    bool
	pendingAdHoc     bool
	sshControlPath   string
	pendingSSHAction string
	browserEndpoint  domain.Endpoint
	browserEntries   []browser.Entry
	browserCursor    int
	browserCurrent   string
	browserHidden    bool
	history          []rsyncengine.Result
	historyCursor    int
	historyDetail    bool
	historySkipped   int
	historyError     string
	dashboard        dashboardLoadedMsg
	hoverKind        string
	hoverIndex       int
}

// New returns an initialized terminal user-interface model.
func New(store *config.Store, settings config.Settings, version string) Model {
	design := newDesignSystem(settings, os.Getenv("NO_COLOR") != "")
	input := textinput.New()
	input.Prompt = "› "
	input.SetWidth(64)
	input.SetVirtualCursor(true)
	spin := newActivitySpinner(design, settings.Motion)
	return Model{
		store:      store,
		settings:   settings,
		design:     design,
		translator: i18n.New(settings.Language),
		version:    version,
		screen:     screenHome,
		input:      input,
		spinner:    spin,
		dryRun:     true,
	}
}

// Init starts initial background work for the terminal interface.
func (m Model) Init() tea.Cmd {
	return loadDashboard(m.store)
}

// Update applies a Bubble Tea message to the model.
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.SetWidth(max(24, min(72, msg.Width-12)))
		return m, nil
	case spinner.TickMsg:
		if m.screen != screenRunning || m.settings.Motion == "none" {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case runEventMsg:
		if m.logOffset > 0 {
			m.logOffset++
		}
		event := rsyncengine.Event(msg)
		m.logLines = append(m.logLines, event.Message)
		if len(m.logLines) > 200 {
			m.logLines = m.logLines[len(m.logLines)-200:]
		}
		return m, waitForRunEvent(m.runEvents)
	case runFinishedMsg:
		m.lastOutcome = msg.outcome
		m.lastErr = msg.err
		m.screen = screenResult
		m.cancel = nil
		return m, loadDashboard(m.store)
	case sshReadyMsg:
		if msg.err != nil {
			if m.pendingSSHAction == "browse" {
				m.status = msg.err.Error()
				m.screen = screenWizard
				return m, nil
			}
			m.lastErr = msg.err
			m.lastOutcome = job.Outcome{}
			m.screen = screenResult
			return m, nil
		}
		if m.pendingSSHAction == "browse" {
			return m, loadRemoteBrowser(m.browserEndpoint, m.sshControlPath, m.browserCurrent, m.browserHidden)
		}
		return m.prepareSudo()
	case sudoReadyMsg:
		if msg.err != nil {
			m.lastErr = msg.err
			m.lastOutcome = job.Outcome{}
			m.screen = screenResult
			return m, nil
		}
		return m.startRun(m.pendingProfile, m.pendingDryRun, m.pendingAdHoc)
	case browserLoadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.screen = screenWizard
			return m, nil
		}
		m.browserEntries = msg.entries
		m.browserCursor = 0
		m.screen = screenBrowser
		return m, nil
	case dashboardLoadedMsg:
		m.dashboard = msg
		return m, nil
	case tea.MouseWheelMsg:
		if m.screen == screenRunning {
			delta := -3
			if msg.Mouse().Button == tea.MouseWheelUp {
				delta = 3
			}
			m.scrollRunLog(delta)
			return m, nil
		}
		code := tea.KeyDown
		if msg.Mouse().Button == tea.MouseWheelUp {
			code = tea.KeyUp
		}
		return m.handleKey(tea.KeyPressMsg(tea.Key{Code: code}))
	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg)
	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)
	case tea.KeyPressMsg:
		if m.screen == screenWizard && (m.wizardStage == wizardSource || m.wizardStage == wizardDestination) && msg.String() == "ctrl+b" {
			return m.openBrowser()
		}
		if m.screen == screenWizard && wizardInputStage(m.wizardStage) && msg.String() != "enter" && msg.String() != "esc" && msg.String() != "ctrl+c" {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m.handleKey(msg)
	}
	if m.screen == screenWizard && wizardInputStage(m.wizardStage) {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(message)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	value := key.String()
	if value == "ctrl+c" {
		if m.screen == screenRunning && m.cancel != nil {
			m.cancel()
			m.status = m.translator.T("status.cancelling")
			return m, nil
		}
		return m, tea.Quit
	}
	switch m.screen {
	case screenHome:
		return m.handleHome(value)
	case screenWizard:
		return m.handleWizard(value)
	case screenProfiles:
		return m.handleProfiles(value)
	case screenRunning:
		switch value {
		case "up", "k":
			m.scrollRunLog(1)
		case "down", "j":
			m.scrollRunLog(-1)
		case "q":
			if m.cancel != nil {
				m.cancel()
				m.status = m.translator.T("status.cancelling")
			}
		}
	case screenResult, screenInfo:
		if value == "q" || value == "esc" || value == "enter" {
			m.screen = screenHome
			m.status = ""
			m.cursor = 0
		}
	case screenSettings:
		return m.handleSettings(value)
	case screenHistory:
		return m.handleHistory(value)
	case screenBrowser:
		return m.handleBrowser(value)
	}
	return m, nil
}

func (m Model) handleHome(key string) (tea.Model, tea.Cmd) {
	items := m.homeItems()
	switch key {
	case "up", "k":
		m.cursor = (m.cursor - 1 + len(items)) % len(items)
	case "down", "j":
		m.cursor = (m.cursor + 1) % len(items)
	case "l":
		next := m.settings
		next.Language = toggledLanguage(m.translator.Language)
		m = m.saveSettings(next)
		if m.settings.Language == next.Language {
			m.status = m.translator.T("status.language")
		}
	case "q":
		return m, tea.Quit
	case "enter":
		switch m.cursor {
		case 0:
			m.startWizard()
			return m, m.input.Focus()
		case 1:
			m.profiles, _ = m.store.ListProfiles()
			m.cursor = 0
			m.screen = screenProfiles
		case 2:
			m.showSnapshotInfo()
		case 3:
			m.showScheduleInfo()
		case 4:
			m.openHistory()
		case 5:
			m.screen = screenSettings
			m.settingsCursor = 0
			m.status = ""
		case 6:
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) handleSettings(key string) (tea.Model, tea.Cmd) {
	const settingsCount = 9
	switch key {
	case "esc", "q":
		m.screen = screenHome
		m.status = ""
	case "up", "k":
		m.settingsCursor = (m.settingsCursor - 1 + settingsCount) % settingsCount
	case "down", "j":
		m.settingsCursor = (m.settingsCursor + 1) % settingsCount
	case "left", "h":
		m = m.changeSetting(-1)
	case "right", "l", "space", "enter":
		m = m.changeSetting(1)
	}
	return m, nil
}

func (m Model) changeSetting(direction int) Model {
	next := m.settings
	switch m.settingsCursor {
	case 0:
		next.Language = cycleSetting(next.Language, []string{"auto", "de", "en"}, direction)
	case 1:
		next.Theme = cycleSetting(next.Theme, []string{"material-dark", "material-light", "midnight", "high-contrast", "no-color"}, direction)
	case 2:
		next.Accent = cycleSetting(next.Accent, []string{"indigo", "blue", "teal", "green", "amber", "rose", "violet"}, direction)
	case 3:
		next.Density = cycleSetting(next.Density, []string{"comfortable", "compact"}, direction)
	case 4:
		next.Icons = cycleSetting(next.Icons, []string{"unicode", "nerd-font"}, direction)
	case 5:
		next.Motion = cycleSetting(next.Motion, []string{"none", "subtle", "expressive"}, direction)
	case 6:
		next.AutoUpdate = !next.AutoUpdate
	case 7:
		next.UpdateChannel = cycleSetting(next.UpdateChannel, []string{"stable", "beta"}, direction)
	case 8:
		next.CheckHours = cycleIntSetting(next.CheckHours, []int{0, 1, 6, 12, 24, 168}, direction)
	}
	return m.saveSettings(next)
}

func (m Model) saveSettings(next config.Settings) Model {
	if err := m.store.SaveSettings(next); err != nil {
		m.status = m.translator.T("settings.save_error", err)
		return m
	}
	m.settings = next
	m.translator = i18n.New(next.Language)
	m.design = newDesignSystem(next, os.Getenv("NO_COLOR") != "")
	m.spinner = newActivitySpinner(m.design, next.Motion)
	m.status = m.translator.T("settings.saved")
	return m
}

func newActivitySpinner(design designSystem, motion string) spinner.Model {
	activity := spinner.Dot
	if motion == "expressive" {
		activity.FPS /= 2
	}
	result := spinner.New(spinner.WithSpinner(activity))
	result.Style = design.Title
	return result
}

func toggledLanguage(language string) string {
	if language == "de" {
		return "en"
	}
	return "de"
}

func cycleSetting(current string, options []string, direction int) string {
	index := 0
	for candidateIndex, option := range options {
		if option == current {
			index = candidateIndex
			break
		}
	}
	index = (index + direction + len(options)) % len(options)
	return options[index]
}

func cycleIntSetting(current int, options []int, direction int) int {
	index := 0
	for candidateIndex, option := range options {
		if option == current {
			index = candidateIndex
			break
		}
	}
	index = (index + direction + len(options)) % len(options)
	return options[index]
}

func (m *Model) startWizard() {
	m.draft = domain.NewProfile(domain.DefaultAdHocName)
	m.wizardStage = wizardChooseStorage
	m.saveProfile = false
	m.profileChoice = 0
	m.wizardName = ""
	m.modeCursor = 0
	m.advancedCursor = 0
	m.expertOptions = nil
	m.dryRun = true
	m.confirm = 0
	m.input.Reset()
	m.input.Blur()
	m.screen = screenWizard
}

func (m Model) handleWizard(key string) (tea.Model, tea.Cmd) {
	if key == "esc" {
		if m.wizardStage > wizardChooseStorage {
			m.wizardStage--
			m.status = ""
			if wizardInputStage(m.wizardStage) {
				m.loadWizardInput()
				return m, m.input.Focus()
			}
			m.input.Blur()
			if m.wizardStage == wizardMode {
				modes := wizardModes(m.saveProfile)
				if m.modeCursor >= len(modes) {
					m.modeCursor = 0
				}
			}
			return m, nil
		}
		m.screen = screenHome
		return m, nil
	}
	if m.wizardStage == wizardChooseStorage {
		switch key {
		case "up", "k", "left", "h":
			m.profileChoice = (m.profileChoice + 1) % 2
		case "down", "j", "right", "l":
			m.profileChoice = (m.profileChoice + 1) % 2
		case "enter":
			m.saveProfile = m.profileChoice == 1
			m.modeCursor = 0
			m.wizardStage = wizardName
			m.loadWizardInput()
			return m, m.input.Focus()
		}
		return m, nil
	}
	if m.wizardStage == wizardName || m.wizardStage == wizardSource || m.wizardStage == wizardDestination {
		if key == "enter" {
			if err := m.acceptWizardInput(); err != nil {
				m.status = err.Error()
				return m, nil
			}
			m.status = ""
			m.wizardStage++
			if wizardInputStage(m.wizardStage) {
				m.loadWizardInput()
				return m, m.input.Focus()
			}
			m.input.Blur()
			return m, nil
		}
		return m, nil
	}
	if m.wizardStage == wizardMode {
		modes := wizardModes(m.saveProfile)
		switch key {
		case "up", "k":
			m.modeCursor = (m.modeCursor - 1 + len(modes)) % len(modes)
		case "down", "j":
			m.modeCursor = (m.modeCursor + 1) % len(modes)
		case "enter":
			m.draft.Mode = modes[m.modeCursor]
			m.wizardStage = wizardAdvanced
		}
		return m, nil
	}
	if m.wizardStage == wizardAdvanced {
		options := wizardAdvancedOptions()
		switch key {
		case "up", "k":
			m.advancedCursor = (m.advancedCursor - 1 + len(options)) % len(options)
		case "down", "j":
			m.advancedCursor = (m.advancedCursor + 1) % len(options)
		case "space":
			m.toggleAdvancedOption(options[m.advancedCursor])
		case "enter":
			m.wizardStage = wizardExpert
			m.loadWizardInput()
			return m, m.input.Focus()
		}
		return m, nil
	}
	if m.wizardStage == wizardExpert {
		if key == "enter" {
			options, err := parseOptionString(m.input.Value())
			if err != nil {
				m.status = err.Error()
				return m, nil
			}
			m.expertOptions = options
			m.draft.Options = append(m.selectedAdvancedOptions(), m.expertOptions...)
			if _, err := rsyncengine.Build(m.draft, rsyncengine.BuildOptions{DryRun: true}); err != nil {
				m.status = err.Error()
				return m, nil
			}
			m.status = ""
			m.input.Blur()
			m.wizardStage = wizardReview
		}
		return m, nil
	}
	switch key {
	case "d":
		m.dryRun = !m.dryRun
		m.confirm = 0
	case "s":
		if m.draft.SourceSemantics == domain.CopyContents {
			m.draft.SourceSemantics = domain.CopyDirectory
		} else {
			m.draft.SourceSemantics = domain.CopyContents
		}
	case "enter":
		if m.draft.Destructive() && !m.dryRun && m.confirm < 1 {
			m.confirm++
			m.status = m.translator.T("wizard.confirm_danger")
			return m, nil
		}
		if err := m.persistWizardProfile(); err != nil {
			m.status = err.Error()
			return m, nil
		}
		return m.beginRun(m.draft, m.dryRun, !m.saveProfile)
	}
	return m, nil
}

func (m Model) persistWizardProfile() error {
	if !m.saveProfile {
		return nil
	}
	return m.store.SaveProfile(m.draft)
}

func (m *Model) acceptWizardInput() error {
	value := strings.TrimSpace(m.input.Value())
	switch m.wizardStage {
	case wizardName:
		if m.saveProfile && value == "" {
			return fmt.Errorf("%s", m.translator.T("wizard.name_required"))
		}
		m.wizardName = value
		if value == "" {
			m.draft.Name = domain.DefaultAdHocName
		} else {
			m.draft.Name = value
		}
	case wizardSource:
		endpoint, err := domain.ParseEndpoint(value)
		if err != nil {
			return err
		}
		m.draft.Source = endpoint
	case wizardDestination:
		endpoint, err := domain.ParseEndpoint(value)
		if err != nil {
			return err
		}
		m.draft.Destination = endpoint
	}
	return nil
}

func (m *Model) loadWizardInput() {
	m.input.Reset()
	switch m.wizardStage {
	case wizardName:
		if m.saveProfile {
			m.input.Placeholder = m.translator.T("wizard.name.profile_placeholder")
		} else {
			m.input.Placeholder = m.translator.T("wizard.name.optional_placeholder")
		}
		m.input.SetValue(m.wizardName)
	case wizardSource:
		m.input.Placeholder = "/source or user@host:/path"
		m.input.SetValue(endpointString(m.draft.Source))
	case wizardDestination:
		m.input.Placeholder = "/destination or ssh://user@host:22/path"
		m.input.SetValue(endpointString(m.draft.Destination))
	case wizardExpert:
		m.input.Placeholder = "--checksum --bwlimit=20m"
		m.input.SetValue(strings.Join(m.expertOptions, " "))
	}
}

func wizardInputStage(stage wizardStage) bool {
	return stage == wizardName || stage == wizardSource || stage == wizardDestination || stage == wizardExpert
}

func (m *Model) toggleAdvancedOption(option string) {
	for index, value := range m.draft.Options {
		if value == option {
			m.draft.Options = append(m.draft.Options[:index], m.draft.Options[index+1:]...)
			return
		}
	}
	m.draft.Options = append(m.draft.Options, option)
}

func (m Model) selectedAdvancedOptions() []string {
	known := optionSet(wizardAdvancedOptions())
	var selected []string
	for _, option := range m.draft.Options {
		if known[option] {
			selected = append(selected, option)
		}
	}
	return selected
}

func (m Model) handleProfiles(key string) (tea.Model, tea.Cmd) {
	if key == "esc" || key == "q" {
		m.screen = screenHome
		m.cursor = 0
		return m, nil
	}
	if len(m.profiles) == 0 {
		return m, nil
	}
	switch key {
	case "up", "k":
		m.cursor = (m.cursor - 1 + len(m.profiles)) % len(m.profiles)
	case "down", "j":
		m.cursor = (m.cursor + 1) % len(m.profiles)
	case "enter":
		profile := m.profiles[m.cursor]
		return m.beginRun(profile, profile.DryRunByDefault, false)
	}
	return m, nil
}

func (m Model) beginRun(profile domain.Profile, dryRun, adHoc bool) (tea.Model, tea.Cmd) {
	m.pendingProfile = profile
	m.pendingDryRun = dryRun
	m.pendingAdHoc = adHoc
	if endpoint, remote := sshclient.RemoteEndpoint(profile); remote {
		controlPath, err := sshclient.ControlPath(m.store.Paths.StateDir, endpoint)
		if err != nil {
			m.lastErr = err
			m.screen = screenResult
			return m, nil
		}
		m.sshControlPath = controlPath
		m.pendingSSHAction = "run"
		m.selected = profile
		m.screen = screenRunning
		m.status = m.translator.T("status.ssh_auth")
		command := sshclient.MasterCommand(endpoint, controlPath)
		return m, tea.ExecProcess(command, func(err error) tea.Msg {
			return sshReadyMsg{err: err}
		})
	}
	return m.prepareSudo()
}

func (m Model) prepareSudo() (tea.Model, tea.Cmd) {
	if m.pendingProfile.UseSudo {
		m.selected = m.pendingProfile
		m.screen = screenRunning
		m.status = m.translator.T("status.sudo_auth")
		return m, tea.ExecProcess(exec.Command("sudo", "-v"), func(err error) tea.Msg {
			return sudoReadyMsg{err: err}
		})
	}
	return m.startRun(m.pendingProfile, m.pendingDryRun, m.pendingAdHoc)
}

func (m Model) startRun(profile domain.Profile, dryRun, adHoc bool) (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.screen = screenRunning
	m.selected = profile
	m.logLines = nil
	m.logOffset = 0
	m.status = ""
	m.runEvents = make(chan tea.Msg, 128)
	manager := job.New(m.store)
	events := m.runEvents
	version := m.version
	go func() {
		outcome, err := manager.Execute(ctx, profile, job.Options{
			DryRun:         dryRun,
			Version:        version,
			SSHControlPath: m.sshControlPath,
			AdHoc:          adHoc,
			OnEvent: func(event rsyncengine.Event) {
				events <- runEventMsg(event)
			},
		})
		events <- runFinishedMsg{outcome: outcome, err: err}
	}()
	if m.settings.Motion == "none" {
		return m, waitForRunEvent(m.runEvents)
	}
	return m, tea.Batch(m.spinner.Tick, waitForRunEvent(m.runEvents))
}

func waitForRunEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-events
	}
}

// View renders the current terminal interface.
func (m Model) View() tea.View {
	content := m.render()
	view := tea.NewView(content)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "rsync-tui"
	return view
}

func (m Model) render() string {
	return m.renderApplication()
}

func (m Model) renderHome() string {
	width := max(34, m.width-8)
	return m.renderDashboard(width, responsiveLayout(m.width))
}

func (m Model) renderSettings() string {
	values := []string{
		m.languageSettingLabel(),
		m.translator.T("settings.theme." + m.settings.Theme),
		m.translator.T("settings.accent." + m.settings.Accent),
		m.translator.T("settings.density." + m.settings.Density),
		m.translator.T("settings.icons." + m.settings.Icons),
		m.translator.T("settings.motion." + m.settings.Motion),
		m.translator.T(fmt.Sprintf("settings.bool.%t", m.settings.AutoUpdate)),
		m.translator.T("settings.channel." + m.settings.UpdateChannel),
		m.checkIntervalLabel(),
	}
	labels := m.settingLabels()
	appearance := m.renderSettingRows(labels, values, 0, 6)
	updates := m.renderSettingRows(labels, values, 6, len(labels))
	preview := m.renderAppearanceSegments()
	body := m.design.Headline.Render(m.design.Icons.Settings+"  "+m.translator.T("settings.title")) + "\n" +
		m.design.Subtitle.Render(m.translator.T("settings.appearance.subtitle")) + "\n\n" +
		m.design.CardHigh.Render(m.design.CardTitle.Render(m.translator.T("settings.appearance"))+"\n\n"+appearance+"\n\n"+preview) + "\n\n" +
		m.design.Card.Render(m.design.CardTitle.Render(m.translator.T("settings.behavior"))+"\n\n"+updates) + "\n\n" +
		m.design.Subtitle.Render(m.translator.T("settings.config", m.store.Paths.ConfigDir)) + "\n" +
		m.design.Subtitle.Render(m.translator.T("settings.help"))
	return body
}

func (m Model) renderAppearanceSegments() string {
	options := []string{m.settings.Theme}
	current := m.settings.Theme
	prefix := "settings.theme."
	switch m.settingsCursor {
	case 2:
		options = []string{"indigo", "blue", "teal", "green", "amber", "rose", "violet"}
		current = m.settings.Accent
		prefix = "settings.accent."
	case 3:
		options = []string{"comfortable", "compact"}
		current = m.settings.Density
		prefix = "settings.density."
	case 4:
		options = []string{"unicode", "nerd-font"}
		current = m.settings.Icons
		prefix = "settings.icons."
	case 5:
		options = []string{"none", "subtle", "expressive"}
		current = m.settings.Motion
		prefix = "settings.motion."
	default:
		options = []string{"material-dark", "material-light", "midnight", "high-contrast", "no-color"}
	}
	labels := make([]string, 0, len(options))
	active := 0
	for index, option := range options {
		labels = append(labels, m.translator.T(prefix+option))
		if option == current {
			active = index
		}
	}
	return m.design.segmentedControl(labels, active, -1, max(28, min(76, m.width-24)))
}

func (m Model) renderSettingRows(labels, values []string, start, end int) string {
	lines := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		line := fmt.Sprintf("%-22s %s", labels[index]+":", values[index])
		if index == m.settingsCursor {
			lines = append(lines, m.design.Selected.Render("› "+line))
		} else if m.isHovered("setting", index) {
			lines = append(lines, m.design.Hover.Render("  "+line))
		} else {
			lines = append(lines, m.design.Item.Render("  "+line))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) checkIntervalLabel() string {
	if m.settings.CheckHours == 0 {
		return m.translator.T("settings.every_start")
	}
	return m.translator.T("settings.hours", m.settings.CheckHours)
}

func (m Model) languageSettingLabel() string {
	if m.settings.Language == "auto" {
		return m.translator.T("settings.language.auto", m.translator.T("settings.language."+m.translator.Language))
	}
	return m.translator.T("settings.language." + m.settings.Language)
}

func (m Model) renderWizard() string {
	stepTitle := func(label string) string {
		return m.design.Title.Render(m.translator.T("wizard.step", int(m.wizardStage)+1, 8, label))
	}
	if m.wizardStage == wizardChooseStorage {
		labels := []string{
			m.translator.T("wizard.storage.one_time"),
			m.translator.T("wizard.storage.profile"),
		}
		hovered := -1
		if m.hoverKind == "wizard-choice" {
			hovered = m.hoverIndex
		}
		content := m.design.segmentedControl(labels, m.profileChoice, hovered, max(24, min(72, m.width-20))) + "\n\n" +
			m.design.Subtitle.Render(m.translator.T("wizard.storage.help"))
		return m.renderWizardFrame(stepTitle(m.translator.T("wizard.storage.title")), content)
	}
	if m.wizardStage == wizardName || m.wizardStage == wizardSource || m.wizardStage == wizardDestination {
		label := m.translator.T("wizard.name.title")
		if m.wizardStage == wizardSource {
			label = m.translator.T("wizard.source")
		}
		if m.wizardStage == wizardDestination {
			label = m.translator.T("wizard.destination")
		}
		help := ""
		if m.wizardStage == wizardSource || m.wizardStage == wizardDestination {
			help = "\n\n" + m.translator.T("wizard.browse_help")
		}
		content := m.design.Field.Render(m.input.View()) + help
		if m.status != "" {
			content += "\n\n" + m.renderStatus(m.status)
		}
		return m.renderWizardFrame(stepTitle(label), content)
	}
	if m.wizardStage == wizardMode {
		var labels []string
		for _, mode := range wizardModes(m.saveProfile) {
			labels = append(labels, m.translator.T("wizard.mode."+string(mode)))
		}
		hovered := -1
		if m.hoverKind == "wizard-mode" {
			hovered = m.hoverIndex
		}
		return m.renderWizardFrame(
			stepTitle(m.translator.T("wizard.mode.title")),
			m.design.segmentedControl(labels, m.modeCursor, hovered, max(24, min(72, m.width-20))),
		)
	}
	if m.wizardStage == wizardAdvanced {
		var lines []string
		selected := optionSet(m.selectedAdvancedOptions())
		for index, option := range wizardAdvancedOptions() {
			mark := "[ ]"
			if selected[option] {
				mark = "[x]"
			}
			label := mark + " " + option
			if index == m.advancedCursor {
				lines = append(lines, m.design.Selected.Render("› "+label))
			} else if m.isHovered("wizard-advanced", index) {
				lines = append(lines, m.design.Hover.Render("  "+label))
			} else {
				lines = append(lines, m.design.Item.Render("  "+label))
			}
		}
		content := strings.Join(lines, "\n") + "\n\n" +
			m.design.Shortcut.Render(m.translator.T("wizard.advanced.help"))
		return m.renderWizardFrame(stepTitle(m.translator.T("wizard.advanced.title")), content)
	}
	if m.wizardStage == wizardExpert {
		content := m.design.Field.Render(m.input.View()) + "\n\n" +
			m.design.Subtitle.Render(m.translator.T("wizard.expert.help"))
		if m.status != "" {
			content += "\n\n" + m.renderStatus(m.status)
		}
		return m.renderWizardFrame(stepTitle(m.translator.T("wizard.expert.title")), content)
	}
	command, err := rsyncengine.Build(m.draft, rsyncengine.BuildOptions{DryRun: m.dryRun})
	commandText := ""
	if err != nil {
		commandText = m.design.Error.Render(err.Error())
	} else {
		commandText = m.design.Body.Render(command.Display)
	}
	danger := ""
	if m.draft.Destructive() {
		danger = "\n" + m.design.Warning.Render(m.translator.T("wizard.warning.destructive"))
	}
	confirm := ""
	if m.confirm > 0 {
		confirm = m.translator.T("wizard.confirm_again")
	}
	storage := m.translator.T("wizard.storage.one_time")
	action := m.translator.T("wizard.action.run")
	if m.saveProfile {
		storage = m.translator.T("wizard.storage.profile")
		action = m.translator.T("wizard.action.save_run")
	}
	displayName := m.draft.Name
	if !m.saveProfile && m.wizardName == "" {
		displayName = m.translator.T("history.one_time")
	}
	summary := fmt.Sprintf("%s: %s\n%s: %s\n%s → %s\n%s: %s\n%s: %s\n%s: %s%s",
		m.translator.T("wizard.name.title"),
		displayName,
		m.translator.T("wizard.storage.title"),
		storage,
		m.draft.Source.Address(false),
		m.draft.Destination.Address(false),
		m.translator.T("wizard.mode.title"),
		m.draft.Mode,
		m.translator.T("wizard.source_semantics"),
		m.draft.SourceSemantics,
		m.translator.T("wizard.dry_run"),
		m.translator.T(fmt.Sprintf("settings.bool.%t", m.dryRun)),
		danger)
	content := m.design.CardHigh.Render(summary) + "\n\n" +
		m.design.Card.Render(m.design.CardTitle.Render(m.translator.T("history.command"))+"\n"+commandText) + "\n\n" +
		m.design.Shortcut.Render(m.translator.T("wizard.review.help", action))
	if confirm != "" {
		content += "\n\n" + m.design.dialog(
			m.translator.T("wizard.warning.destructive"),
			confirm,
			max(24, min(72, m.width-20)),
			true,
		)
	}
	if m.status != "" {
		content += "\n\n" + m.renderStatus(m.status)
	}
	return m.renderWizardFrame(stepTitle(m.translator.T("wizard.review.title")), content)
}

func (m Model) renderWizardFrame(title, content string) string {
	return m.renderWizardStepper() + "\n\n" +
		title + "\n\n" +
		m.design.CardHigh.Render(content)
}

func (m Model) renderWizardStepper() string {
	if m.width < 76 {
		return m.design.chip(fmt.Sprintf("%d / 8", int(m.wizardStage)+1), true)
	}
	labels := []string{
		m.translator.T("wizard.storage.title"),
		m.translator.T("wizard.name.title"),
		m.translator.T("wizard.source"),
		m.translator.T("wizard.destination"),
		m.translator.T("wizard.mode.title"),
		m.translator.T("wizard.advanced.title"),
		m.translator.T("wizard.expert.title"),
		m.translator.T("wizard.review.title"),
	}
	steps := make([]string, 0, len(labels))
	for index, label := range labels {
		mark := "○"
		style := m.design.Subtitle
		if index < int(m.wizardStage) {
			mark = m.design.Icons.Success
			style = m.design.Success
		}
		if index == int(m.wizardStage) {
			mark = "●"
			style = m.design.Title
		}
		steps = append(steps, style.Render(mark+" "+truncateDisplay(label, 12)))
	}
	return strings.Join(steps, "  ")
}

func (m Model) renderProfiles() string {
	if len(m.profiles) == 0 {
		return m.design.Headline.Render(m.design.Icons.Profiles+"  "+m.translator.T("menu.profiles")) + "\n\n" +
			m.design.emptyState(m.design.Icons.Profiles, m.translator.T("status.no_profiles"), m.translator.T("profiles.subtitle")) + "\n\n" +
			m.design.Shortcut.Render(m.translator.T("help.back"))
	}
	var cards []string
	for index, profile := range m.profiles {
		title := fmt.Sprintf("%s  %s", m.design.Icons.Profiles, profile.Name)
		body := fmt.Sprintf("%s  ·  %s → %s",
			m.translator.T("wizard.mode."+string(profile.Mode)),
			profile.Source.Address(false),
			profile.Destination.Address(false))
		cards = append(cards, m.design.card(title, body, 0, index == m.cursor, m.isHovered("profile", index), false))
	}
	return m.design.Headline.Render(m.design.Icons.Profiles+"  "+m.translator.T("menu.profiles")) + "\n" +
		m.design.Subtitle.Render(m.translator.T("profiles.subtitle")) + "\n\n" +
		strings.Join(cards, "\n\n") + "\n\n" +
		m.design.Shortcut.Render(m.translator.T("profiles.help"))
}

func (m Model) renderRunning() string {
	limit := max(2, m.height-22)
	maxOffset := max(0, len(m.logLines)-limit)
	offset := min(maxOffset, max(0, m.logOffset))
	end := len(m.logLines) - offset
	start := max(0, end-limit)
	lines := m.logLines[start:end]
	if offset > 0 {
		lines = append([]string{m.translator.T("running.scrolled", offset)}, lines...)
	}
	name := m.selected.Name
	if m.pendingAdHoc && name == domain.DefaultAdHocName {
		name = m.translator.T("history.one_time")
	}
	indicator := "●"
	if m.settings.Motion != "none" {
		indicator = m.spinner.View()
	}
	progress := m.renderProgressIndicator(max(20, min(72, m.width-18)))
	header := m.design.CardPrimary.Render(fmt.Sprintf("%s  %s\n%s",
		indicator,
		m.translator.T("page.running"),
		name))
	logBody := strings.Join(lines, "\n")
	if strings.TrimSpace(logBody) == "" {
		logBody = m.translator.T("running.waiting")
	}
	cancelLabel := "Ctrl+C  " + m.translator.T("action.cancel")
	cancelStyle := m.design.ChipPrimary
	if m.isHovered("cancel", 0) {
		cancelLabel = "› " + cancelLabel
		cancelStyle = m.design.Selected
	}
	return header + "\n\n" +
		progress + "\n\n" +
		m.design.CardHigh.Render(m.design.CardTitle.Render(m.translator.T("running.activity"))+"\n\n"+logBody) + "\n\n" +
		cancelStyle.Render(cancelLabel)
}

func (m *Model) scrollRunLog(delta int) {
	limit := max(2, m.height-22)
	m.logOffset = min(max(0, len(m.logLines)-limit), max(0, m.logOffset+delta))
}

func (m Model) renderResult() string {
	style := m.design.Success
	icon := m.design.Icons.Success
	title := m.translator.T("result.success")
	if m.lastErr != nil {
		style = m.design.Error
		icon = m.design.Icons.Error
		title = m.translator.T("result.failure")
	}
	result := m.lastOutcome.Result
	summary := []string{
		fmt.Sprintf("%s: %s", m.translator.T("history.name"), m.historyEntryName(result)),
		fmt.Sprintf("%s: %s", m.translator.T("history.mode"), m.historyModeLabel(result.Mode)),
		fmt.Sprintf("%s: %s", m.translator.T("history.duration"), formatHistoryDuration(result)),
		fmt.Sprintf("%s: %d", m.translator.T("history.exit_code"), result.ExitCode),
	}
	if result.Source != "" || result.Destination != "" {
		summary = append(summary, fmt.Sprintf("%s → %s", result.Source, result.Destination))
	}
	if result.Error != "" {
		summary = append(summary, "", m.design.Error.Render(result.Error))
	}
	if len(m.lastOutcome.NotificationWarnings) > 0 {
		summary = append(summary, "", m.design.Warning.Render(
			m.translator.T("result.notification_warnings", len(m.lastOutcome.NotificationWarnings))))
	}
	return m.design.CardPrimary.Render(style.Render(icon+"  "+title)) + "\n\n" +
		m.design.CardHigh.Render(strings.Join(summary, "\n")) + "\n\n" +
		m.design.Shortcut.Render(m.translator.T("result.help"))
}

func (m Model) renderBrowser() string {
	var cards []string
	limit := max(5, m.height-16)
	start := 0
	if m.browserCursor >= limit {
		start = m.browserCursor - limit + 1
	}
	end := min(len(m.browserEntries), start+limit)
	for index := start; index < end; index++ {
		entry := m.browserEntries[index]
		label := m.design.Icons.Folder + "  " + entry.Name + "/"
		cards = append(cards, m.design.card(label, entry.Path, 0, index == m.browserCursor, m.isHovered("browser", index), false))
	}
	if len(cards) == 0 {
		cards = append(cards, m.design.emptyState(m.design.Icons.Folder, m.translator.T("browser.empty"), ""))
	}
	endpoint := m.design.Icons.Folder
	if m.browserEndpoint.IsRemote() {
		endpoint = m.design.Icons.Remote
	}
	chips := m.design.chip(endpoint+"  "+m.browserCurrent, true) + " " +
		m.design.chip(m.translator.T(fmt.Sprintf("settings.bool.%t", m.browserHidden))+" "+m.translator.T("browser.hidden"), false)
	return m.design.Headline.Render(m.translator.T("page.browser")) + "\n\n" +
		chips + "\n\n" +
		strings.Join(cards, "\n") + "\n\n" +
		m.design.Shortcut.Render(m.translator.T("browser.help"))
}

func (m Model) renderProgressIndicator(width int) string {
	width = max(10, width)
	if m.settings.Motion == "none" {
		return m.design.Divider.Render(strings.Repeat("━", width))
	}
	position := len(m.logLines) % max(1, width-5)
	activeWidth := 5
	if m.settings.Motion == "expressive" {
		activeWidth = 9
	}
	bar := strings.Repeat("─", position) + strings.Repeat("━", activeWidth)
	if remaining := width - lipgloss.Width(bar); remaining > 0 {
		bar += strings.Repeat("─", remaining)
	}
	return m.design.Title.Render(bar)
}

func (m Model) openBrowser() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		home, _ := os.UserHomeDir()
		value = home
	}
	endpoint, err := domain.ParseEndpoint(value)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.browserEndpoint = endpoint
	m.browserCurrent = endpoint.Path
	m.browserCursor = 0
	if !endpoint.IsRemote() {
		entries, err := browser.LocalDirectories(endpoint.Path, m.browserHidden)
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.browserEntries = entries
		m.screen = screenBrowser
		return m, nil
	}
	controlPath, err := sshclient.ControlPath(m.store.Paths.StateDir, endpoint)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.sshControlPath = controlPath
	m.pendingSSHAction = "browse"
	m.screen = screenRunning
	m.status = m.translator.T("status.ssh_auth")
	command := sshclient.MasterCommand(endpoint, controlPath)
	return m, tea.ExecProcess(command, func(err error) tea.Msg {
		return sshReadyMsg{err: err}
	})
}

func (m Model) handleBrowser(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.screen = screenWizard
		m.status = ""
		return m, m.input.Focus()
	case "up", "k":
		if len(m.browserEntries) > 0 {
			m.browserCursor = (m.browserCursor - 1 + len(m.browserEntries)) % len(m.browserEntries)
		}
	case "down", "j":
		if len(m.browserEntries) > 0 {
			m.browserCursor = (m.browserCursor + 1) % len(m.browserEntries)
		}
	case "s":
		m.browserEndpoint.Path = m.browserCurrent
		if m.wizardStage == wizardSource {
			m.draft.Source = m.browserEndpoint
		} else {
			m.draft.Destination = m.browserEndpoint
		}
		m.input.SetValue(endpointString(m.browserEndpoint))
		m.screen = screenWizard
		return m, m.input.Focus()
	case "h":
		m.browserHidden = !m.browserHidden
		return m.reloadBrowser()
	case "enter":
		if len(m.browserEntries) == 0 {
			return m, nil
		}
		m.browserCurrent = m.browserEntries[m.browserCursor].Path
		return m.reloadBrowser()
	}
	return m, nil
}

func (m Model) reloadBrowser() (tea.Model, tea.Cmd) {
	if m.browserEndpoint.IsRemote() {
		return m, loadRemoteBrowser(m.browserEndpoint, m.sshControlPath, m.browserCurrent, m.browserHidden)
	}
	entries, err := browser.LocalDirectories(m.browserCurrent, m.browserHidden)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.browserEntries = entries
	m.browserCursor = 0
	return m, nil
}

func loadRemoteBrowser(endpoint domain.Endpoint, controlPath, current string, showHidden bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		entries, err := browser.RemoteDirectories(ctx, endpoint, controlPath, current, showHidden)
		return browserLoadedMsg{entries: entries, err: err}
	}
}

func (m *Model) showSnapshotInfo() {
	profiles, _ := m.store.ListProfiles()
	var lines []string
	for _, profile := range profiles {
		if profile.Mode == domain.ModeSnapshot {
			lines = append(lines, "• "+profile.Name+" — "+profile.Destination.Path)
		}
	}
	if len(lines) == 0 {
		lines = append(lines, m.translator.T("snapshot.empty"))
	}
	m.status = strings.Join(lines, "\n")
	m.screen = screenInfo
}

func (m *Model) showScheduleInfo() {
	profiles, _ := m.store.ListProfiles()
	var lines []string
	for _, profile := range profiles {
		if profile.Schedule.Enabled {
			lines = append(lines, fmt.Sprintf("• %s — %s", profile.Name, profile.Schedule.OnCalendar))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, m.translator.T("schedule.empty"))
	}
	m.status = strings.Join(lines, "\n")
	m.screen = screenInfo
}

func (m *Model) openHistory() {
	history, err := rsyncengine.LoadHistory(m.store.Paths.StateDir, 100)
	m.history = history.Entries
	m.historySkipped = history.Skipped
	m.historyCursor = 0
	m.historyDetail = false
	m.historyError = ""
	if err != nil {
		m.historyError = err.Error()
	}
	m.status = ""
	m.screen = screenHistory
}

func (m Model) handleHistory(key string) (tea.Model, tea.Cmd) {
	if m.historyDetail {
		switch key {
		case "esc", "enter":
			m.historyDetail = false
		case "q":
			m.historyDetail = false
			m.screen = screenHome
		}
		return m, nil
	}
	switch key {
	case "esc", "q":
		m.screen = screenHome
	case "up", "k":
		if len(m.history) > 0 {
			m.historyCursor = (m.historyCursor - 1 + len(m.history)) % len(m.history)
		}
	case "down", "j":
		if len(m.history) > 0 {
			m.historyCursor = (m.historyCursor + 1) % len(m.history)
		}
	case "enter":
		if len(m.history) > 0 {
			m.historyDetail = true
		}
	}
	return m, nil
}

func (m Model) renderHistory() string {
	if m.historyError != "" {
		return m.design.Title.Render(m.translator.T("history.title")) + "\n\n" +
			m.design.Error.Render(m.historyError) + "\n\n" +
			m.design.Subtitle.Render(m.translator.T("history.help.back"))
	}
	if len(m.history) == 0 {
		return m.design.Title.Render(m.translator.T("history.title")) + "\n\n" +
			m.design.Subtitle.Render(m.translator.T("history.empty")) + "\n\n" +
			m.design.Subtitle.Render(m.translator.T("history.help.back"))
	}
	if m.historyDetail {
		return m.renderHistoryDetail(m.history[m.historyCursor])
	}

	visible := max(1, (m.height-12)/3)
	start := 0
	if m.historyCursor >= visible {
		start = m.historyCursor - visible + 1
	}
	end := min(len(m.history), start+visible)
	width := max(30, min(92, m.width-14))
	cards := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		entry := m.history[index]
		name := m.historyEntryName(entry)
		mode := m.historyModeLabel(entry.Mode)
		status := "✓"
		statusStyle := m.design.Success
		if entry.ExitCode != 0 {
			status = "✗"
			statusStyle = m.design.Error
		}
		dryRun := ""
		if entry.DryRun {
			dryRun = " · " + m.translator.T("history.dry_run")
		}
		header := fmt.Sprintf("%s  %s [%s] · %s%s",
			formatHistoryTime(entry.StartedAt),
			name,
			mode,
			formatHistoryDuration(entry),
			dryRun)
		pathLine := entry.Source + " → " + entry.Destination
		if entry.Source == "" && entry.Destination == "" {
			pathLine = m.translator.T("history.legacy_entry")
		}
		header = statusStyle.Render(status) + "  " + truncateDisplay(header, width-4)
		pathLine = truncateDisplay(pathLine, width)
		cards = append(cards, m.design.card(header, pathLine, 0, index == m.historyCursor, m.isHovered("history", index), false))
	}
	footer := m.translator.T("history.help.list")
	if m.historySkipped > 0 {
		footer = m.translator.T("history.skipped", m.historySkipped) + "\n" + footer
	}
	return m.design.Headline.Render(m.design.Icons.History+"  "+m.translator.T("history.title")) + "\n\n" +
		strings.Join(cards, "\n") + "\n\n" +
		m.design.Subtitle.Render(footer)
}

func (m Model) renderHistoryDetail(entry rsyncengine.Result) string {
	status := m.translator.T("history.status.success")
	statusText := m.design.Success.Render(status)
	if entry.ExitCode != 0 {
		status = m.translator.T("history.status.failure")
		statusText = m.design.Error.Render(status)
	}
	lines := []string{
		fmt.Sprintf("%s: %s", m.translator.T("history.name"), m.historyEntryName(entry)),
		fmt.Sprintf("%s: %s", m.translator.T("history.status"), statusText),
		fmt.Sprintf("%s: %s", m.translator.T("history.started"), formatHistoryTime(entry.StartedAt)),
		fmt.Sprintf("%s: %s", m.translator.T("history.finished"), formatHistoryTime(entry.FinishedAt)),
		fmt.Sprintf("%s: %s", m.translator.T("history.duration"), formatHistoryDuration(entry)),
		fmt.Sprintf("%s: %d", m.translator.T("history.exit_code"), entry.ExitCode),
		fmt.Sprintf("%s: %s", m.translator.T("history.dry_run"), m.translator.T(fmt.Sprintf("settings.bool.%t", entry.DryRun))),
	}
	if entry.Mode != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", m.translator.T("history.mode"), m.historyModeLabel(entry.Mode)))
	}
	if entry.Source != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", m.translator.T("history.source"), entry.Source))
	}
	if entry.Destination != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", m.translator.T("history.destination"), entry.Destination))
	}
	if entry.Error != "" {
		lines = append(lines, "", m.design.Error.Render(m.translator.T("history.error")+": "+entry.Error))
	}
	if entry.Command != "" {
		commandWidth := max(28, min(90, m.width-16))
		lines = append(lines, "",
			m.translator.T("history.command")+":",
			lipgloss.NewStyle().Width(commandWidth).Render(entry.Command))
	}
	return m.design.Headline.Render(m.translator.T("history.detail_title")) + "\n\n" +
		m.design.CardHigh.Render(strings.Join(lines, "\n")) + "\n\n" +
		m.design.Subtitle.Render(m.translator.T("history.help.detail"))
}

func (m Model) historyEntryName(entry rsyncengine.Result) string {
	if entry.AdHoc && (entry.ProfileName == "" || entry.ProfileName == domain.DefaultAdHocName) {
		return m.translator.T("history.one_time")
	}
	if strings.TrimSpace(entry.ProfileName) == "" {
		return m.translator.T("history.unnamed")
	}
	return entry.ProfileName
}

func (m Model) historyModeLabel(mode domain.Mode) string {
	switch mode {
	case domain.ModeCopy, domain.ModeMirror, domain.ModeMove, domain.ModeSnapshot, domain.ModeRestore, domain.ModeCustom:
		return m.translator.T("wizard.mode." + string(mode))
	default:
		return m.translator.T("history.mode.unknown")
	}
}

func formatHistoryTime(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	return value.Local().Format("02.01.2006 15:04:05")
}

func formatHistoryDuration(entry rsyncengine.Result) string {
	if entry.StartedAt.IsZero() || entry.FinishedAt.IsZero() || entry.FinishedAt.Before(entry.StartedAt) {
		return "—"
	}
	duration := entry.FinishedAt.Sub(entry.StartedAt)
	if duration < time.Second {
		return duration.Round(time.Millisecond).String()
	}
	return duration.Round(time.Second).String()
}

func truncateDisplay(value string, width int) string {
	if width <= 1 {
		return "…"
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func (m Model) isHovered(kind string, index int) bool {
	return m.hoverKind == kind && m.hoverIndex == index
}

func (m Model) homeItems() []string {
	return []string{
		m.translator.T("menu.new"),
		m.translator.T("menu.profiles"),
		m.translator.T("menu.snapshots"),
		m.translator.T("menu.schedules"),
		m.translator.T("menu.history"),
		m.translator.T("menu.settings"),
		m.translator.T("menu.quit"),
	}
}

func wizardModes(saveProfile bool) []domain.Mode {
	modes := []domain.Mode{domain.ModeCopy, domain.ModeMirror, domain.ModeMove}
	if saveProfile {
		modes = append(modes, domain.ModeSnapshot)
	}
	return append(modes, domain.ModeRestore, domain.ModeCustom)
}

func wizardAdvancedOptions() []string {
	return []string{
		"--checksum",
		"--compress",
		"--acls",
		"--xattrs",
		"--hard-links",
		"--numeric-ids",
		"--one-file-system",
		"--sparse",
		"--fsync",
		"--ignore-existing",
		"--size-only",
		"--whole-file",
		"--mkpath",
		"--itemize-changes",
	}
}

func optionSet(options []string) map[string]bool {
	result := make(map[string]bool, len(options))
	for _, option := range options {
		result[option] = true
	}
	return result
}

func parseOptionString(value string) ([]string, error) {
	var result []string
	var current strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if current.Len() > 0 {
			result = append(result, current.String())
			current.Reset()
		}
	}
	for _, character := range value {
		if escaped {
			current.WriteRune(character)
			escaped = false
			continue
		}
		if character == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				current.WriteRune(character)
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if character == ' ' || character == '\t' || character == '\n' {
			flush()
			continue
		}
		current.WriteRune(character)
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("unterminated quote or escape in expert arguments")
	}
	flush()
	return result, nil
}

func endpointString(endpoint domain.Endpoint) string {
	if endpoint.Path == "" {
		return ""
	}
	if endpoint.IsRemote() {
		return endpoint.SSHHost() + ":" + endpoint.Path
	}
	return endpoint.Path
}

func (m Model) renderStatus(status string) string {
	if status == "" {
		return ""
	}
	return m.design.Error.Render(status)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
