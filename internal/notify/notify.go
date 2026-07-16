package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

// Event contains the profile and result data sent to notification backends.
type Event struct {
	Event            string    `json:"event"`
	Status           string    `json:"status"`
	ProfileID        string    `json:"profile_id"`
	ProfileName      string    `json:"profile_name"`
	StartedAt        time.Time `json:"started_at"`
	FinishedAt       time.Time `json:"finished_at"`
	DurationSeconds  float64   `json:"duration_seconds"`
	TransferredBytes int64     `json:"transferred_bytes,omitempty"`
	ChangedFiles     int       `json:"changed_files,omitempty"`
	DeletedFiles     int       `json:"deleted_files,omitempty"`
	ExitCode         int       `json:"exit_code"`
	Message          string    `json:"message,omitempty"`
	Version          string    `json:"version"`
}

// Sender dispatches job-completion notifications.
type Sender struct {
	Client *http.Client
}

// Send delivers an event through each configured notification backend.
func (s Sender) Send(ctx context.Context, configuration domain.Notifications, event Event) []error {
	if event.Status == "success" && !configuration.OnSuccess {
		return nil
	}
	if event.Status != "success" && !configuration.OnFailure {
		return nil
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	var errs []error
	if token, err := secretValue(configuration.NtfyToken, configuration.NtfyTokenEnv, configuration.NtfyTokenFile, "ntfy token"); err != nil {
		errs = append(errs, err)
		configuration.NtfyURL = ""
	} else {
		configuration.NtfyToken = token
	}
	if token, err := secretValue(configuration.GotifyToken, configuration.GotifyTokenEnv, configuration.GotifyTokenFile, "Gotify token"); err != nil {
		errs = append(errs, err)
		configuration.GotifyURL = ""
	} else {
		configuration.GotifyToken = token
	}
	if configuration.NtfyURL != "" {
		if err := sendNtfy(ctx, client, configuration, event); err != nil {
			errs = append(errs, redact(err, configuration))
		}
	}
	if configuration.GotifyURL != "" {
		if err := sendGotify(ctx, client, configuration, event); err != nil {
			errs = append(errs, redact(err, configuration))
		}
	}
	if configuration.WebhookURL != "" {
		if err := sendJSON(ctx, client, configuration.WebhookURL, nil, event); err != nil {
			errs = append(errs, redact(err, configuration))
		}
	}
	if configuration.Sendmail != "" {
		if err := sendMailCommand(ctx, configuration.Sendmail, configuration.SMTP, event); err != nil {
			errs = append(errs, redact(err, configuration))
		}
	}
	if configuration.SMTP.Address != "" {
		if err := sendSMTP(configuration.SMTP, event); err != nil {
			errs = append(errs, redact(err, configuration))
		}
	}
	return errs
}

func sendNtfy(ctx context.Context, client *http.Client, configuration domain.Notifications, event Event) error {
	headers := map[string]string{
		"Title":    "rsync-tui: " + event.ProfileName,
		"Priority": priority(event.Status),
		"Tags":     tag(event.Status),
	}
	if configuration.NtfyToken != "" {
		headers["Authorization"] = "Bearer " + configuration.NtfyToken
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, configuration.NtfyURL, strings.NewReader(summary(event)))
	if err != nil {
		return err
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	return do(client, request)
}

func sendGotify(ctx context.Context, client *http.Client, configuration domain.Notifications, event Event) error {
	endpoint := strings.TrimRight(configuration.GotifyURL, "/") + "/message"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	query := parsed.Query()
	query.Set("token", configuration.GotifyToken)
	parsed.RawQuery = query.Encode()
	body := map[string]any{
		"title":    "rsync-tui: " + event.ProfileName,
		"message":  summary(event),
		"priority": gotifyPriority(event.Status),
	}
	return sendJSON(ctx, client, parsed.String(), nil, body)
}

func sendJSON(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	for key, header := range headers {
		request.Header.Set(key, header)
	}
	return do(client, request)
}

func do(client *http.Client, request *http.Request) error {
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("notification endpoint returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func sendMailCommand(ctx context.Context, command string, configuration domain.SMTPConfig, event Event) error {
	if strings.ContainsAny(command, "\r\n") {
		return errors.New("sendmail path contains a newline")
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return err
	}
	message := mailMessage(configuration, event)
	process := exec.CommandContext(ctx, path, "-t", "-i")
	process.Stdin = strings.NewReader(message)
	output, err := process.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sendmail: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func sendSMTP(configuration domain.SMTPConfig, event Event) error {
	host, _, err := net.SplitHostPort(configuration.Address)
	if err != nil {
		return err
	}
	password, err := smtpPassword(configuration)
	if err != nil {
		return err
	}
	var auth smtp.Auth
	if configuration.Username != "" {
		auth = smtp.PlainAuth("", configuration.Username, password, host)
	}
	connection, err := smtp.Dial(configuration.Address)
	if err != nil {
		return err
	}
	defer connection.Close()
	if ok, _ := connection.Extension("STARTTLS"); ok {
		if err := connection.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := connection.Auth(auth); err != nil {
			return err
		}
	}
	if err := connection.Mail(configuration.From); err != nil {
		return err
	}
	if err := connection.Rcpt(configuration.To); err != nil {
		return err
	}
	writer, err := connection.Data()
	if err != nil {
		return err
	}
	if _, err := io.WriteString(writer, mailMessage(configuration, event)); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func smtpPassword(configuration domain.SMTPConfig) (string, error) {
	if configuration.PasswordEnv != "" {
		value, ok := os.LookupEnv(configuration.PasswordEnv)
		if !ok {
			return "", fmt.Errorf("SMTP password environment variable %s is not set", configuration.PasswordEnv)
		}
		return value, nil
	}
	if configuration.PasswordFile != "" {
		info, err := os.Stat(configuration.PasswordFile)
		if err != nil {
			return "", err
		}
		if info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf("SMTP password file %s must not be group/world accessible", filepath.Base(configuration.PasswordFile))
		}
		data, err := os.ReadFile(configuration.PasswordFile)
		return strings.TrimSpace(string(data)), err
	}
	return "", nil
}

func secretValue(direct, environment, file, label string) (string, error) {
	if environment != "" {
		value, ok := os.LookupEnv(environment)
		if !ok {
			return "", fmt.Errorf("%s environment variable %s is not set", label, environment)
		}
		return strings.TrimSpace(value), nil
	}
	if file != "" {
		info, err := os.Stat(file)
		if err != nil {
			return "", err
		}
		if info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf("%s file %s must not be group/world accessible", label, filepath.Base(file))
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return direct, nil
}

func mailMessage(configuration domain.SMTPConfig, event Event) string {
	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: rsync-tui: %s — %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		configuration.From, configuration.To, event.ProfileName, event.Status, summary(event))
}

func summary(event Event) string {
	message := fmt.Sprintf("%s: %s (exit %d, %.0fs)", event.ProfileName, event.Status, event.ExitCode, event.DurationSeconds)
	if event.Message != "" {
		message += "\n" + event.Message
	}
	return message
}

func priority(status string) string {
	if status == "success" {
		return "default"
	}
	return "high"
}

func tag(status string) string {
	if status == "success" {
		return "white_check_mark"
	}
	return "warning"
}

func gotifyPriority(status string) int {
	if status == "success" {
		return 2
	}
	return 8
}

func redact(err error, configuration domain.Notifications) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	for _, secret := range []string{configuration.NtfyToken, configuration.GotifyToken, configuration.NtfyURL, configuration.GotifyURL, configuration.WebhookURL} {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "[redacted]")
		}
	}
	return errors.New(message)
}
