package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/fabianschmeltzer/rsync-tui/internal/config"
)

type colorRoles struct {
	Background         string
	Surface            string
	SurfaceHigh        string
	SurfaceVariant     string
	Primary            string
	OnPrimary          string
	PrimaryContainer   string
	OnPrimaryContainer string
	OnSurface          string
	OnSurfaceVariant   string
	Outline            string
	Success            string
	Warning            string
	Error              string
}

type iconSet struct {
	App       string
	Home      string
	New       string
	Profiles  string
	Snapshots string
	Schedules string
	History   string
	Settings  string
	Quit      string
	Back      string
	Success   string
	Error     string
	Info      string
	Folder    string
	Remote    string
}

type designSystem struct {
	Roles       colorRoles
	Icons       iconSet
	Theme       string
	Accent      string
	Density     string
	Motion      string
	NoColor     bool
	Padding     int
	Gap         int
	Background  lipgloss.Style
	AppBar      lipgloss.Style
	Title       lipgloss.Style
	Headline    lipgloss.Style
	Body        lipgloss.Style
	Subtitle    lipgloss.Style
	Selected    lipgloss.Style
	Hover       lipgloss.Style
	Item        lipgloss.Style
	Panel       lipgloss.Style
	Card        lipgloss.Style
	CardHigh    lipgloss.Style
	CardPrimary lipgloss.Style
	CardTitle   lipgloss.Style
	Chip        lipgloss.Style
	ChipPrimary lipgloss.Style
	Field       lipgloss.Style
	Divider     lipgloss.Style
	Warning     lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	Snackbar    lipgloss.Style
	Shortcut    lipgloss.Style
}

func newDesignSystem(settings config.Settings, forceNoColor bool) designSystem {
	noColor := forceNoColor || settings.Theme == "no-color"
	roles := materialRoles(settings.Theme, settings.Accent)
	padding := 2
	gap := 2
	if settings.Density == "compact" {
		padding = 1
		gap = 1
	}
	design := designSystem{
		Roles:   roles,
		Icons:   materialIcons(settings.Icons),
		Theme:   settings.Theme,
		Accent:  settings.Accent,
		Density: settings.Density,
		Motion:  settings.Motion,
		NoColor: noColor,
		Padding: padding,
		Gap:     gap,
	}
	if noColor {
		design.Background = lipgloss.NewStyle()
		design.AppBar = lipgloss.NewStyle()
		design.Title = lipgloss.NewStyle()
		design.Headline = lipgloss.NewStyle()
		design.Body = lipgloss.NewStyle()
		design.Subtitle = lipgloss.NewStyle()
		design.Selected = lipgloss.NewStyle().Padding(0, 1)
		design.Hover = lipgloss.NewStyle().Padding(0, 1)
		design.Item = lipgloss.NewStyle().Padding(0, 1)
		vertical := 1
		if settings.Density == "compact" {
			vertical = 0
		}
		design.Panel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(vertical, padding)
		design.Card = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(vertical, padding)
		design.CardHigh = design.Card
		design.CardPrimary = design.Card
		design.CardTitle = lipgloss.NewStyle()
		design.Chip = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		design.ChipPrimary = design.Chip
		design.Field = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		design.Divider = lipgloss.NewStyle()
		design.Warning = lipgloss.NewStyle()
		design.Error = lipgloss.NewStyle()
		design.Success = lipgloss.NewStyle()
		design.Snackbar = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
		design.Shortcut = lipgloss.NewStyle()
		return design
	}

	color := lipgloss.Color
	vertical := 1
	if settings.Density == "compact" {
		vertical = 0
	}
	design.Background = lipgloss.NewStyle().
		Background(color(roles.Background)).
		Foreground(color(roles.OnSurface))
	design.AppBar = lipgloss.NewStyle().
		Background(color(roles.Surface)).
		Foreground(color(roles.OnSurface)).
		Bold(true).
		Padding(0, 2)
	design.Title = lipgloss.NewStyle().
		Foreground(color(roles.Primary)).
		Bold(true)
	design.Headline = lipgloss.NewStyle().
		Foreground(color(roles.OnSurface)).
		Bold(true)
	design.Body = lipgloss.NewStyle().Foreground(color(roles.OnSurface))
	design.Subtitle = lipgloss.NewStyle().Foreground(color(roles.OnSurfaceVariant))
	design.Selected = lipgloss.NewStyle().
		Background(color(roles.PrimaryContainer)).
		Foreground(color(roles.OnPrimaryContainer)).
		Bold(true).
		Padding(0, 1)
	design.Hover = lipgloss.NewStyle().
		Background(color(roles.SurfaceHigh)).
		Foreground(color(roles.OnSurface)).
		Padding(0, 1)
	design.Item = lipgloss.NewStyle().
		Foreground(color(roles.OnSurface)).
		Padding(0, 1)
	design.Panel = lipgloss.NewStyle().
		Background(color(roles.Surface)).
		Foreground(color(roles.OnSurface)).
		Padding(vertical, padding)
	design.Card = lipgloss.NewStyle().
		Background(color(roles.Surface)).
		Foreground(color(roles.OnSurface)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color(roles.Outline)).
		Padding(vertical, padding)
	design.CardHigh = lipgloss.NewStyle().
		Background(color(roles.SurfaceHigh)).
		Foreground(color(roles.OnSurface)).
		Padding(vertical, padding)
	design.CardPrimary = lipgloss.NewStyle().
		Background(color(roles.PrimaryContainer)).
		Foreground(color(roles.OnPrimaryContainer)).
		Bold(true).
		Padding(vertical, padding)
	design.CardTitle = lipgloss.NewStyle().
		Foreground(color(roles.OnSurface)).
		Bold(true)
	design.Chip = lipgloss.NewStyle().
		Background(color(roles.SurfaceVariant)).
		Foreground(color(roles.OnSurfaceVariant)).
		Padding(0, 1)
	design.ChipPrimary = lipgloss.NewStyle().
		Background(color(roles.PrimaryContainer)).
		Foreground(color(roles.OnPrimaryContainer)).
		Bold(true).
		Padding(0, 1)
	design.Field = lipgloss.NewStyle().
		Background(color(roles.SurfaceHigh)).
		Foreground(color(roles.OnSurface)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color(roles.Outline)).
		Padding(0, 1)
	design.Divider = lipgloss.NewStyle().Foreground(color(roles.Outline))
	design.Warning = lipgloss.NewStyle().Foreground(color(roles.Warning)).Bold(true)
	design.Error = lipgloss.NewStyle().Foreground(color(roles.Error)).Bold(true)
	design.Success = lipgloss.NewStyle().Foreground(color(roles.Success)).Bold(true)
	design.Snackbar = lipgloss.NewStyle().
		Background(color(roles.OnSurface)).
		Foreground(color(roles.Background)).
		Padding(0, 2)
	design.Shortcut = lipgloss.NewStyle().Foreground(color(roles.OnSurfaceVariant))
	return design
}

func materialRoles(theme, accent string) colorRoles {
	if theme == "auto" || theme == "" {
		theme = "material-dark"
	}
	primary, container, onContainer := accentColors(accent, theme == "material-light")
	switch theme {
	case "material-light":
		return colorRoles{
			Background:         "#F9F9FF",
			Surface:            "#F0F1FA",
			SurfaceHigh:        "#E4E6F0",
			SurfaceVariant:     "#DEE1EC",
			Primary:            primary,
			OnPrimary:          "#FFFFFF",
			PrimaryContainer:   container,
			OnPrimaryContainer: onContainer,
			OnSurface:          "#1A1B20",
			OnSurfaceVariant:   "#45464F",
			Outline:            "#767680",
			Success:            "#146C2E",
			Warning:            "#8A5100",
			Error:              "#BA1A1A",
		}
	case "midnight":
		return colorRoles{
			Background:         "#070B18",
			Surface:            "#0E1528",
			SurfaceHigh:        "#17213A",
			SurfaceVariant:     "#202B46",
			Primary:            primary,
			OnPrimary:          "#FFFFFF",
			PrimaryContainer:   container,
			OnPrimaryContainer: onContainer,
			OnSurface:          "#E7E9F4",
			OnSurfaceVariant:   "#BFC5D9",
			Outline:            "#737B94",
			Success:            "#72DA8C",
			Warning:            "#FFB95F",
			Error:              "#FFB4AB",
		}
	case "high-contrast":
		return colorRoles{
			Background:         "#000000",
			Surface:            "#0A0A0A",
			SurfaceHigh:        "#1B1B1B",
			SurfaceVariant:     "#262626",
			Primary:            "#FFF200",
			OnPrimary:          "#000000",
			PrimaryContainer:   "#FFF200",
			OnPrimaryContainer: "#000000",
			OnSurface:          "#FFFFFF",
			OnSurfaceVariant:   "#F2F2F2",
			Outline:            "#FFFFFF",
			Success:            "#59FF88",
			Warning:            "#FFD166",
			Error:              "#FF6B6B",
		}
	default:
		return colorRoles{
			Background:         "#111318",
			Surface:            "#191C20",
			SurfaceHigh:        "#24272D",
			SurfaceVariant:     "#2B303B",
			Primary:            primary,
			OnPrimary:          "#FFFFFF",
			PrimaryContainer:   container,
			OnPrimaryContainer: onContainer,
			OnSurface:          "#E2E2E9",
			OnSurfaceVariant:   "#C4C6D0",
			Outline:            "#8E9099",
			Success:            "#72DA8C",
			Warning:            "#FFB95F",
			Error:              "#FFB4AB",
		}
	}
}

func accentColors(accent string, light bool) (primary, container, onContainer string) {
	type pair struct {
		darkPrimary    string
		darkContainer  string
		darkOn         string
		lightPrimary   string
		lightContainer string
		lightOn        string
	}
	palettes := map[string]pair{
		"blue":   {"#A9C7FF", "#274777", "#D7E3FF", "#365F9D", "#D7E3FF", "#001B3E"},
		"teal":   {"#80D5CF", "#00504C", "#9CF2EB", "#006A65", "#9CF2EB", "#00201E"},
		"green":  {"#9CD49F", "#22512C", "#B8F0B9", "#386A3F", "#BAF2BC", "#002108"},
		"amber":  {"#F8BD48", "#5D4200", "#FFDEA1", "#765A00", "#FFDF9F", "#261A00"},
		"rose":   {"#FFB1C8", "#6D394A", "#FFD9E2", "#8C4F60", "#FFD9E2", "#3A071E"},
		"violet": {"#D0BCFF", "#4F378B", "#EADDFF", "#6750A4", "#EADDFF", "#21005D"},
		"indigo": {"#BBC3FF", "#3E477D", "#DEE0FF", "#555E9C", "#DFE0FF", "#111A4B"},
	}
	selected, ok := palettes[accent]
	if !ok {
		selected = palettes["indigo"]
	}
	if light {
		return selected.lightPrimary, selected.lightContainer, selected.lightOn
	}
	return selected.darkPrimary, selected.darkContainer, selected.darkOn
}

func materialIcons(mode string) iconSet {
	if mode == "nerd-font" {
		return iconSet{
			App:       "󰁯",
			Home:      "󰋜",
			New:       "󰐕",
			Profiles:  "󰈙",
			Snapshots: "󰆓",
			Schedules: "󰥔",
			History:   "󰋚",
			Settings:  "󰒓",
			Quit:      "󰗼",
			Back:      "󰁍",
			Success:   "󰄬",
			Error:     "󰅙",
			Info:      "󰋼",
			Folder:    "󰉋",
			Remote:    "󰢹",
		}
	}
	return iconSet{
		App:       "◆",
		Home:      "⌂",
		New:       "＋",
		Profiles:  "▤",
		Snapshots: "◫",
		Schedules: "◷",
		History:   "↶",
		Settings:  "⚙",
		Quit:      "⏻",
		Back:      "←",
		Success:   "✓",
		Error:     "✕",
		Info:      "●",
		Folder:    "▰",
		Remote:    "⇄",
	}
}

func (d designSystem) chip(label string, primary bool) string {
	if primary {
		return d.ChipPrimary.Render(label)
	}
	return d.Chip.Render(label)
}

func (d designSystem) segmentedControl(labels []string, active, hovered, width int) string {
	var lines []string
	line := ""
	for index, label := range labels {
		style := d.Chip
		marker := "  "
		if index == hovered {
			style = d.Hover
		}
		if index == active {
			style = d.ChipPrimary
			marker = "● "
		}
		segment := style.Render(marker + label)
		next := segment
		if line != "" {
			next = line + " " + segment
		}
		if width > 0 && line != "" && lipgloss.Width(next) > width {
			lines = append(lines, line)
			line = segment
		} else {
			line = next
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (d designSystem) dialog(title, body string, width int, danger bool) string {
	style := d.CardHigh.Copy().Border(lipgloss.DoubleBorder())
	titleStyle := d.Title
	icon := d.Icons.Info
	if danger {
		titleStyle = d.Error
		icon = d.Icons.Error
		if !d.NoColor {
			style = style.BorderForeground(lipgloss.Color(d.Roles.Error))
		}
	}
	content := titleStyle.Render(icon + "  " + title)
	if strings.TrimSpace(body) != "" {
		content += "\n\n" + d.Body.Render(body)
	}
	if width > 0 {
		style = style.Width(width)
	}
	return style.Render(content)
}

func (d designSystem) emptyState(icon, title, body string) string {
	content := d.Title.Render(icon + "  " + title)
	if strings.TrimSpace(body) != "" {
		content += "\n" + d.Subtitle.Render(body)
	}
	return d.CardHigh.Render(content)
}

func (d designSystem) card(title, body string, width int, selected, hovered, primary bool) string {
	style := d.Card
	if primary {
		style = d.CardPrimary
	}
	if hovered {
		if primary {
			style = style.Copy().Border(lipgloss.RoundedBorder())
		} else {
			style = d.CardHigh.Copy().Border(lipgloss.RoundedBorder())
		}
		if !d.NoColor {
			style = style.BorderForeground(lipgloss.Color(d.Roles.Outline))
		}
	}
	if selected {
		style = style.Copy().Border(lipgloss.DoubleBorder())
		if !d.NoColor {
			style = style.BorderForeground(lipgloss.Color(d.Roles.Primary))
		}
	}
	marker := ""
	if d.NoColor {
		if selected {
			marker = "› "
		} else if hovered {
			marker = "· "
		}
	}
	content := d.CardTitle.Render(marker + title)
	if strings.TrimSpace(body) != "" {
		content += "\n" + d.Subtitle.Render(body)
	}
	if width > 0 {
		style = style.Width(width)
	}
	return style.Render(content)
}

func (d designSystem) shortcutBar(help string, width int) string {
	content := d.Shortcut.Render(help)
	if width > 0 {
		return lipgloss.NewStyle().Width(width).Render(content)
	}
	return content
}

func (d designSystem) snackbar(message string, width int) string {
	if strings.TrimSpace(message) == "" {
		return ""
	}
	content := d.Snackbar.Render(fmt.Sprintf("%s  %s", d.Icons.Info, message))
	if width > 0 && lipgloss.Width(content) > width {
		return truncateDisplay(content, width)
	}
	return content
}

type layoutMode int

const (
	layoutCompact layoutMode = iota
	layoutMedium
	layoutExpanded
)

func responsiveLayout(width int) layoutMode {
	switch {
	case width >= 110:
		return layoutExpanded
	case width >= 72:
		return layoutMedium
	default:
		return layoutCompact
	}
}
