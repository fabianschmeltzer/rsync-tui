package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	"github.com/fabianschmeltzer/rsync-tui/internal/i18n"
	"github.com/fabianschmeltzer/rsync-tui/internal/job"
	"github.com/fabianschmeltzer/rsync-tui/internal/notify"
	rsyncengine "github.com/fabianschmeltzer/rsync-tui/internal/rsync"
	"github.com/fabianschmeltzer/rsync-tui/internal/scheduler"
	"github.com/fabianschmeltzer/rsync-tui/internal/snapshot"
	"github.com/fabianschmeltzer/rsync-tui/internal/sshclient"
	"github.com/fabianschmeltzer/rsync-tui/internal/tui"
	updater "github.com/fabianschmeltzer/rsync-tui/internal/update"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		return runTUI()
	}
	switch args[0] {
	case "--version", "version":
		return versionCommand(args[1:])
	case "run":
		return runCommand(args[1:])
	case "doctor":
		return doctorCommand(args[1:])
	case "update":
		return updateCommand(args[1:])
	case "schedule":
		return scheduleCommand(args[1:])
	case "profile":
		return profileCommand(args[1:])
	case "notify":
		return notifyCommand(args[1:])
	case "snapshot":
		return snapshotCommand(args[1:])
	case "self-test":
		return selfTest()
	case "help", "--help", "-h":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printHelp()
		return 64
	}
}

func runTUI() int {
	store, err := config.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		return 1
	}
	settings, err := store.LoadSettings()
	if err != nil {
		fmt.Fprintln(os.Stderr, "settings:", err)
		return 1
	}
	if updated := automaticUpdate(store, settings); updated {
		fmt.Println("rsync-tui was updated. Please start it again.")
		return 0
	}
	model := tui.New(store, settings, version)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui:", err)
		return 1
	}
	return 0
}

func automaticUpdate(store *config.Store, settings config.Settings) bool {
	if version == "dev" || !settings.AutoUpdate {
		return false
	}
	interval := time.Duration(settings.CheckHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	now := time.Now()
	if !updater.Due(store.Paths.StateDir, interval, now) {
		return false
	}
	_ = updater.MarkChecked(store.Paths.StateDir, now)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client := updater.Client{}
	available, err := client.Check(ctx, version, settings.UpdateChannel)
	if err != nil || available == nil {
		return false
	}
	return client.Install(ctx, *available) == nil
}

func versionCommand(args []string) int {
	flags := flag.NewFlagSet("version", flag.ContinueOnError)
	asJSON := flags.Bool("json", false, "print machine-readable version information")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	info := map[string]string{
		"version":    version,
		"commit":     commit,
		"build_date": buildDate,
		"go":         runtime.Version(),
		"platform":   runtime.GOOS + "/" + runtime.GOARCH,
	}
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(info)
	} else {
		fmt.Printf("rsync-tui %s (%s, %s, %s)\n", version, commit, buildDate, info["platform"])
	}
	return 0
}

func runCommand(args []string) int {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	profileID := flags.String("profile", "", "profile ID or name")
	dryRun := flags.Bool("dry-run", false, "force a dry-run")
	scheduled := flags.Bool("scheduled", false, "run with unattended safety rules")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	if *profileID == "" {
		fmt.Fprintln(os.Stderr, "--profile is required")
		return 64
	}
	store, err := config.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 64
	}
	profile, err := store.LoadProfile(*profileID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 64
	}
	manager := job.New(store)
	controlPath := ""
	if endpoint, remote := sshclient.RemoteEndpoint(profile); remote {
		if *scheduled {
			if err := sshclient.BatchCheck(context.Background(), endpoint, ""); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 65
			}
		} else {
			controlPath, err = sshclient.ControlPath(store.Paths.StateDir, endpoint)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 65
			}
			command := sshclient.MasterCommand(endpoint, controlPath)
			command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := command.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "SSH authentication:", err)
				return 65
			}
		}
	}
	if profile.UseSudo && !*scheduled {
		if err := authenticateSudo(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 65
		}
	}
	outcome, err := manager.Execute(context.Background(), profile, job.Options{
		DryRun:         *dryRun,
		Scheduled:      *scheduled,
		Version:        version,
		SSHControlPath: controlPath,
		OnEvent: func(event rsyncengine.Event) {
			fmt.Println(event.Message)
		},
	})
	if len(outcome.NotificationWarnings) > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d notification(s) failed\n", len(outcome.NotificationWarnings))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if outcome.Result.ExitCode > 0 {
			return outcome.Result.ExitCode
		}
		return 1
	}
	return 0
}

func doctorCommand(args []string) int {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	asJSON := flags.Bool("json", false, "print machine-readable diagnostics")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	store, configErr := config.Open()
	type diagnostic struct {
		Name    string `json:"name"`
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	var diagnostics []diagnostic
	for _, binary := range []string{"rsync", "ssh", "sudo", "systemctl"} {
		path, err := exec.LookPath(binary)
		diagnostics = append(diagnostics, diagnostic{Name: binary, OK: err == nil, Message: firstNonEmpty(path, errorString(err))})
	}
	diagnostics = append(diagnostics, diagnostic{Name: "config", OK: configErr == nil, Message: firstNonEmpty(pathString(store), errorString(configErr))})
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(diagnostics)
	} else {
		fmt.Println("rsync-tui system diagnostics")
		for _, item := range diagnostics {
			mark := "✓"
			if !item.OK {
				mark = "✗"
			}
			fmt.Printf("%s %-10s %s\n", mark, item.Name, item.Message)
		}
	}
	for _, item := range diagnostics {
		if !item.OK && (item.Name == "rsync" || item.Name == "config") {
			return 1
		}
	}
	return 0
}

func updateCommand(args []string) int {
	flags := flag.NewFlagSet("update", flag.ContinueOnError)
	checkOnly := flags.Bool("check", false, "only check for an update")
	rollback := flags.Bool("rollback", false, "restore the previous binary")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	if *rollback {
		if err := updater.Rollback(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 73
		}
		fmt.Println("Previous rsync-tui binary restored.")
		return 0
	}
	store, err := config.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 73
	}
	settings, err := store.LoadSettings()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 73
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := updater.Client{}
	available, err := client.Check(ctx, version, settings.UpdateChannel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 73
	}
	if available == nil {
		fmt.Println("rsync-tui is up to date.")
		return 0
	}
	fmt.Println("Update available:", available.Version)
	if *checkOnly {
		return 0
	}
	if err := client.Install(ctx, *available); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 73
	}
	fmt.Println("Update installed. Restart rsync-tui to use it.")
	return 0
}

func scheduleCommand(args []string) int {
	if len(args) == 0 || args[0] != "install" {
		fmt.Fprintln(os.Stderr, "usage: rsync-tui schedule install --profile <id|name>")
		return 64
	}
	flags := flag.NewFlagSet("schedule install", flag.ContinueOnError)
	profileID := flags.String("profile", "", "profile ID or name")
	if err := flags.Parse(args[1:]); err != nil {
		return 64
	}
	store, err := config.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	profile, err := store.LoadProfile(*profileID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 64
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	units, err := scheduler.Install(profile, executable)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Installed %s and %s\n", units.ServiceName, units.TimerName)
	return 0
}

func profileCommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: rsync-tui profile list|show|configure")
		return 64
	}
	store, err := config.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	switch args[0] {
	case "list":
		profiles, err := store.ListProfiles()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(args) > 1 && args[1] == "--json" {
			_ = json.NewEncoder(os.Stdout).Encode(profiles)
			return 0
		}
		for _, profile := range profiles {
			fmt.Printf("%s\t%s\t%s\n", profile.ID, profile.Mode, profile.Name)
		}
		return 0
	case "show":
		flags := flag.NewFlagSet("profile show", flag.ContinueOnError)
		id := flags.String("profile", "", "profile ID or name")
		if err := flags.Parse(args[1:]); err != nil {
			return 64
		}
		profile, err := store.LoadProfile(*id)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 64
		}
		_ = json.NewEncoder(os.Stdout).Encode(profile)
		return 0
	case "configure":
		return configureProfile(store, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown profile command %q\n", args[0])
		return 64
	}
}

func configureProfile(store *config.Store, args []string) int {
	flags := flag.NewFlagSet("profile configure", flag.ContinueOnError)
	id := flags.String("profile", "", "profile ID or name")
	schedule := optionalString{}
	retention := optionalString{}
	ntfyURL := optionalString{}
	ntfyTokenEnv := optionalString{}
	ntfyTokenFile := optionalString{}
	gotifyURL := optionalString{}
	gotifyTokenEnv := optionalString{}
	gotifyTokenFile := optionalString{}
	webhookURL := optionalString{}
	sendmail := optionalString{}
	smtpAddress := optionalString{}
	smtpUsername := optionalString{}
	smtpPasswordEnv := optionalString{}
	smtpPasswordFile := optionalString{}
	smtpFrom := optionalString{}
	smtpTo := optionalString{}
	useSudo := optionalBool{}
	dryRun := optionalBool{}
	systemTimer := optionalBool{}
	allowDestructive := optionalBool{}
	onSuccess := optionalBool{}
	onFailure := optionalBool{}
	flags.Var(&schedule, "schedule", "systemd OnCalendar expression, or 'off'")
	flags.Var(&retention, "retention", "last_n or gfs")
	flags.Var(&ntfyURL, "ntfy-url", "ntfy topic URL")
	flags.Var(&ntfyTokenEnv, "ntfy-token-env", "environment variable containing the ntfy token")
	flags.Var(&ntfyTokenFile, "ntfy-token-file", "0600 file containing the ntfy token")
	flags.Var(&gotifyURL, "gotify-url", "Gotify server URL")
	flags.Var(&gotifyTokenEnv, "gotify-token-env", "environment variable containing the Gotify token")
	flags.Var(&gotifyTokenFile, "gotify-token-file", "0600 file containing the Gotify token")
	flags.Var(&webhookURL, "webhook-url", "generic webhook URL")
	flags.Var(&sendmail, "sendmail", "sendmail-compatible executable")
	flags.Var(&smtpAddress, "smtp-address", "SMTP host:port")
	flags.Var(&smtpUsername, "smtp-username", "SMTP username")
	flags.Var(&smtpPasswordEnv, "smtp-password-env", "environment variable containing the SMTP password")
	flags.Var(&smtpPasswordFile, "smtp-password-file", "0600 file containing the SMTP password")
	flags.Var(&smtpFrom, "smtp-from", "mail sender")
	flags.Var(&smtpTo, "smtp-to", "mail recipient")
	flags.Var(&useSudo, "sudo", "true or false")
	flags.Var(&dryRun, "dry-run-default", "true or false")
	flags.Var(&systemTimer, "system-timer", "true or false")
	flags.Var(&allowDestructive, "allow-unattended-destructive", "true or false")
	flags.Var(&onSuccess, "notify-success", "true or false")
	flags.Var(&onFailure, "notify-failure", "true or false")
	maxDelete := flags.Int("max-delete", -1, "maximum scheduled mirror deletions")
	maxRemovals := flags.Int("max-source-removals", -1, "maximum scheduled Move removals")
	lastN := flags.Int("last-n", -1, "Last-N snapshot count")
	daily := flags.Int("daily", -1, "daily GFS count")
	weekly := flags.Int("weekly", -1, "weekly GFS count")
	monthly := flags.Int("monthly", -1, "monthly GFS count")
	if err := flags.Parse(args); err != nil {
		return 64
	}
	profile, err := store.LoadProfile(*id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 64
	}
	if schedule.set {
		if schedule.value == "off" {
			profile.Schedule.Enabled = false
			profile.Schedule.OnCalendar = ""
		} else {
			profile.Schedule.Enabled = true
			profile.Schedule.OnCalendar = schedule.value
		}
	}
	if retention.set {
		profile.Snapshot.Retention.Mode = domain.RetentionMode(retention.value)
	}
	applyOptionalBool(&profile.UseSudo, useSudo)
	applyOptionalBool(&profile.DryRunByDefault, dryRun)
	applyOptionalBool(&profile.Schedule.System, systemTimer)
	applyOptionalBool(&profile.Safety.AllowUnattendedDestructive, allowDestructive)
	applyOptionalBool(&profile.Notifications.OnSuccess, onSuccess)
	applyOptionalBool(&profile.Notifications.OnFailure, onFailure)
	if *maxDelete >= 0 {
		profile.Safety.MaxDelete = *maxDelete
	}
	if *maxRemovals >= 0 {
		profile.Safety.MaxSourceRemovals = *maxRemovals
	}
	if *lastN >= 0 {
		profile.Snapshot.Retention.LastN = *lastN
	}
	if *daily >= 0 {
		profile.Snapshot.Retention.Daily = *daily
	}
	if *weekly >= 0 {
		profile.Snapshot.Retention.Weekly = *weekly
	}
	if *monthly >= 0 {
		profile.Snapshot.Retention.Monthly = *monthly
	}
	applyOptionalString(&profile.Notifications.NtfyURL, ntfyURL)
	applyOptionalString(&profile.Notifications.NtfyTokenEnv, ntfyTokenEnv)
	applyOptionalString(&profile.Notifications.NtfyTokenFile, ntfyTokenFile)
	applyOptionalString(&profile.Notifications.GotifyURL, gotifyURL)
	applyOptionalString(&profile.Notifications.GotifyTokenEnv, gotifyTokenEnv)
	applyOptionalString(&profile.Notifications.GotifyTokenFile, gotifyTokenFile)
	applyOptionalString(&profile.Notifications.WebhookURL, webhookURL)
	applyOptionalString(&profile.Notifications.Sendmail, sendmail)
	applyOptionalString(&profile.Notifications.SMTP.Address, smtpAddress)
	applyOptionalString(&profile.Notifications.SMTP.Username, smtpUsername)
	applyOptionalString(&profile.Notifications.SMTP.PasswordEnv, smtpPasswordEnv)
	applyOptionalString(&profile.Notifications.SMTP.PasswordFile, smtpPasswordFile)
	applyOptionalString(&profile.Notifications.SMTP.From, smtpFrom)
	applyOptionalString(&profile.Notifications.SMTP.To, smtpTo)
	if err := store.SaveProfile(profile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 64
	}
	fmt.Println("Updated profile", profile.Name)
	return 0
}

func notifyCommand(args []string) int {
	if len(args) == 0 || args[0] != "test" {
		fmt.Fprintln(os.Stderr, "usage: rsync-tui notify test --profile <id|name>")
		return 64
	}
	flags := flag.NewFlagSet("notify test", flag.ContinueOnError)
	id := flags.String("profile", "", "profile ID or name")
	if err := flags.Parse(args[1:]); err != nil {
		return 64
	}
	store, err := config.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	profile, err := store.LoadProfile(*id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 64
	}
	configuration := profile.Notifications
	configuration.OnSuccess = true
	now := time.Now().UTC()
	errs := (notify.Sender{}).Send(context.Background(), configuration, notify.Event{
		Event:       "notification.test",
		Status:      "success",
		ProfileID:   profile.ID,
		ProfileName: profile.Name,
		StartedAt:   now,
		FinishedAt:  now,
		Message:     "rsync-tui test notification",
		Version:     version,
	})
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	fmt.Println("Test notifications sent.")
	return 0
}

func snapshotCommand(args []string) int {
	if len(args) == 0 || (args[0] != "list" && args[0] != "restore") {
		fmt.Fprintln(os.Stderr, "usage: rsync-tui snapshot list|restore --profile <id|name>")
		return 64
	}
	flags := flag.NewFlagSet("snapshot "+args[0], flag.ContinueOnError)
	id := flags.String("profile", "", "snapshot profile ID or name")
	snapshotID := flags.String("snapshot", "", "snapshot ID")
	destination := flags.String("destination", "", "restore destination override")
	execute := flags.Bool("execute", false, "perform the restore instead of a dry-run")
	yes := flags.Bool("yes", false, "confirm a real restore")
	if err := flags.Parse(args[1:]); err != nil {
		return 64
	}
	store, err := config.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	profile, err := store.LoadProfile(*id)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 64
	}
	if profile.Mode != domain.ModeSnapshot {
		fmt.Fprintln(os.Stderr, "profile is not a snapshot profile")
		return 64
	}
	controlPath, err := authenticateProfileRemote(store, profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 65
	}
	manager := snapshot.Manager{}
	var records []snapshot.Record
	if profile.Destination.IsRemote() {
		records, err = manager.ListRemote(context.Background(), profile, controlPath)
	} else {
		records, err = manager.List(profile)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if args[0] == "list" {
		for _, record := range records {
			fmt.Printf("%s\t%s\t%s\n", record.ID, record.CreatedAt.Format(time.RFC3339), record.Path)
		}
		return 0
	}
	if *snapshotID == "" {
		fmt.Fprintln(os.Stderr, "--snapshot is required")
		return 64
	}
	var selected *snapshot.Record
	for index := range records {
		if records[index].ID == *snapshotID {
			selected = &records[index]
			break
		}
	}
	if selected == nil {
		fmt.Fprintln(os.Stderr, "snapshot not found")
		return 64
	}
	if *execute && !*yes {
		fmt.Fprintln(os.Stderr, "a real restore requires both --execute and --yes")
		return 65
	}
	restore := domain.NewProfile("Restore " + profile.Name)
	restore.Mode = domain.ModeRestore
	restore.SourceSemantics = domain.CopyContents
	restore.Source = profile.Destination
	restore.Source.Path = selected.Path
	restore.Destination = profile.Source
	restore.UseSudo = profile.UseSudo
	if *destination != "" {
		restore.Destination = domain.Endpoint{Kind: domain.EndpointLocal, Path: *destination}
	}
	if restore.UseSudo {
		if err := authenticateSudo(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 65
		}
	}
	outcome, err := job.New(store).Execute(context.Background(), restore, job.Options{
		DryRun:         !*execute,
		Version:        version,
		SSHControlPath: controlPath,
		OnEvent: func(event rsyncengine.Event) {
			fmt.Println(event.Message)
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if outcome.Result.ExitCode > 0 {
			return outcome.Result.ExitCode
		}
		return 1
	}
	return 0
}

func authenticateProfileRemote(store *config.Store, profile domain.Profile) (string, error) {
	endpoint, remote := sshclient.RemoteEndpoint(profile)
	if !remote {
		return "", nil
	}
	controlPath, err := sshclient.ControlPath(store.Paths.StateDir, endpoint)
	if err != nil {
		return "", err
	}
	command := sshclient.MasterCommand(endpoint, controlPath)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("SSH authentication: %w", err)
	}
	return controlPath, nil
}

func authenticateSudo() error {
	command := exec.Command("sudo", "-v")
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("sudo authentication: %w", err)
	}
	return nil
}

func selfTest() int {
	if err := i18n.ValidateCatalogs(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	profile := domain.NewProfile("self-test")
	profile.Source.Path = "/tmp/source"
	profile.Destination.Path = "/tmp/destination"
	if _, err := rsyncengine.Build(profile, rsyncengine.BuildOptions{DryRun: true}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("ok")
	return 0
}

func printHelp() {
	fmt.Print(`rsync-tui — safe local and SSH transfers

Usage:
  rsync-tui
  rsync-tui run --profile <id|name> [--dry-run] [--scheduled]
  rsync-tui profile list|show|configure
  rsync-tui notify test --profile <id|name>
  rsync-tui snapshot list|restore --profile <id|name>
  rsync-tui doctor [--json]
  rsync-tui update [--check|--rollback]
  rsync-tui version [--json]
`)
}

type optionalString struct {
	value string
	set   bool
}

func (value *optionalString) String() string { return value.value }
func (value *optionalString) Set(input string) error {
	value.value, value.set = input, true
	return nil
}

type optionalBool struct {
	value bool
	set   bool
}

func (value *optionalBool) String() string { return fmt.Sprint(value.value) }
func (value *optionalBool) Set(input string) error {
	parsed, err := strconv.ParseBool(input)
	if err != nil {
		return err
	}
	value.value, value.set = parsed, true
	return nil
}

func applyOptionalString(target *string, value optionalString) {
	if value.set {
		*target = value.value
	}
}

func applyOptionalBool(target *bool, value optionalBool) {
	if value.set {
		*target = value.value
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "-"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func pathString(store *config.Store) string {
	if store == nil {
		return ""
	}
	return store.Paths.ConfigDir
}
