package rsync

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

type BuildOptions struct {
	DryRun              bool
	DestinationOverride string
	LinkDestination     string
	FrozenFileList      string
	SSHControlPath      string
}

type Command struct {
	Program string   `json:"program"`
	Args    []string `json:"args"`
	Display string   `json:"display"`
}

var forbiddenOptions = []string{
	"--server",
	"--sender",
	"--daemon",
	"--config",
	"--password-file",
}

func Build(profile domain.Profile, options BuildOptions) (Command, error) {
	if err := profile.Validate(); err != nil {
		return Command{}, err
	}

	args := baseArgs(profile.Mode)
	args = append(args,
		"--human-readable",
		"--info=progress2,name1,stats2",
		"--stats",
		"--partial",
		"--partial-dir=.rsync-partial",
	)
	if options.DryRun {
		args = append(args, "--dry-run", "--itemize-changes")
	}
	if profile.Source.IsRemote() || profile.Destination.IsRemote() {
		args = append(args, "--secluded-args")
	}
	if profile.Safety.MaxDelete > 0 && profile.Mode == domain.ModeMirror {
		args = append(args, "--max-delete="+strconv.Itoa(profile.Safety.MaxDelete))
	}
	if options.LinkDestination != "" {
		args = append(args, "--link-dest="+options.LinkDestination)
	}
	if options.FrozenFileList != "" {
		args = append(args, "--from0", "--files-from="+options.FrozenFileList)
	}
	for _, include := range profile.Filters.Include {
		args = append(args, "--include="+include)
	}
	for _, exclude := range profile.Filters.Exclude {
		args = append(args, "--exclude="+exclude)
	}
	for _, option := range profile.Options {
		if err := validateExtraOption(option); err != nil {
			return Command{}, err
		}
		args = append(args, option)
	}
	if err := validateOptionConflicts(args); err != nil {
		return Command{}, err
	}

	if shell := sshShell(profile.Source, profile.Destination, options.SSHControlPath); shell != "" {
		args = append(args, "--rsh="+shell)
	}

	source := profile.Source.Address(profile.SourceSemantics == domain.CopyContents)
	destination := profile.Destination.Address(false)
	if options.DestinationOverride != "" {
		if profile.Destination.IsRemote() {
			destination = profile.Destination.SSHHost() + ":" + options.DestinationOverride
		} else {
			destination = options.DestinationOverride
		}
	}
	args = append(args, "--", source, destination)

	program := "rsync"
	displayParts := []string{shellQuote(program)}
	for _, arg := range args {
		displayParts = append(displayParts, shellQuote(arg))
	}
	return Command{
		Program: program,
		Args:    args,
		Display: strings.Join(displayParts, " "),
	}, nil
}

func baseArgs(mode domain.Mode) []string {
	switch mode {
	case domain.ModeCopy:
		return []string{"--archive", "--update"}
	case domain.ModeMirror:
		return []string{"--archive", "--delete-delay"}
	case domain.ModeMove:
		return []string{"--archive", "--remove-source-files"}
	case domain.ModeSnapshot, domain.ModeRestore:
		return []string{"--archive"}
	case domain.ModeCustom:
		return nil
	default:
		return nil
	}
}

func validateExtraOption(option string) error {
	option = strings.TrimSpace(option)
	if option == "" {
		return errors.New("empty expert option")
	}
	if !strings.HasPrefix(option, "-") {
		return fmt.Errorf("expert option %q must use --option=value form", option)
	}
	for _, forbidden := range forbiddenOptions {
		if option == forbidden || strings.HasPrefix(option, forbidden+"=") {
			return fmt.Errorf("expert option %q is managed or unsupported", option)
		}
	}
	if option == "--" {
		return errors.New("expert options cannot add positional arguments")
	}
	return nil
}

func validateOptionConflicts(options []string) error {
	present := make(map[string]bool)
	for _, option := range options {
		name, _, _ := strings.Cut(option, "=")
		present[name] = true
	}
	conflictGroups := [][]string{
		{"--delete-before", "--delete-during", "--delete-delay", "--delete-after"},
		{"--inplace", "--delay-updates"},
		{"--whole-file", "--no-whole-file"},
		{"--compress", "--no-compress"},
	}
	for _, group := range conflictGroups {
		var selected []string
		for _, option := range group {
			if present[option] {
				selected = append(selected, option)
			}
		}
		if len(selected) > 1 {
			return fmt.Errorf("conflicting rsync options: %s", strings.Join(selected, ", "))
		}
	}
	return nil
}

func sshShell(source, destination domain.Endpoint, controlPath string) string {
	endpoint := source
	if destination.IsRemote() {
		endpoint = destination
	}
	if !endpoint.IsRemote() {
		return ""
	}
	parts := []string{"ssh"}
	if controlPath != "" {
		parts = append(parts, "-o", "ControlPath="+shellQuote(controlPath))
	}
	if endpoint.Port > 0 {
		parts = append(parts, "-p", strconv.Itoa(endpoint.Port))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if !strings.ContainsAny(value, " \t\r\n'\"\\$`!&|;()<>*?[]{}") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func DisplayWithSudo(command Command, nonInteractive bool) string {
	if nonInteractive {
		return "sudo -n " + command.Display
	}
	return "sudo " + command.Display
}
