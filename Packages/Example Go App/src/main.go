package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"Talos/sdk/go/talos"
)

const (
	statusFile     = "example_go_status.json"
	tsExampleAppID = "app.example.ts"
)

// statusPayload is written to scoped storage so the web UI can show live sidecar stats.
type statusPayload struct {
	SchemaVersion       int       `json:"schema_version"`
	AppID               string    `json:"app_id"`
	BootUTC             time.Time `json:"boot_utc"`
	LastTickUTC         time.Time `json:"last_tick_utc"`
	Ticks               int       `json:"ticks"`
	NetGranted          bool      `json:"net_internet_granted"`
	NetMessage          string    `json:"net_internet_message,omitempty"`
	SendMessageNote     string    `json:"send_message_note,omitempty"`
	BroadcastRecipients int32     `json:"last_bootstrap_broadcast_recipients"`
	PrevStateFound      bool      `json:"prev_hub_state_found"`
	PrevStateSnippet    string    `json:"prev_hub_state_snippet,omitempty"`
	HeartbeatPreview    string    `json:"heartbeat_file_preview,omitempty"`
}

type runner struct {
	client *talos.Client
	appID  string
	start  time.Time
	ticks  int

	netGranted       bool
	netMsg           string
	sendMsgNote      string
	bcastN           int32
	prevStateFound   bool
	prevStateSnippet string
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	appID := os.Getenv("TALOS_APP_ID")
	hub := os.Getenv("TALOS_HUB_SOCKET")
	if appID == "" || hub == "" {
		slog.Error("missing required environment", "have_app_id", appID != "", "have_hub_socket", hub != "")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dialCtx, cancelDial := context.WithTimeout(ctx, 5*time.Second)
	client, err := talos.Dial(dialCtx, hub)
	cancelDial()
	if err != nil {
		slog.Error("hub dial failed", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	if err := run(ctx, client, appID); err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("run failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, client *talos.Client, appID string) error {
	r, err := bootstrap(ctx, client, appID)
	if err != nil {
		return err
	}

	if err := r.tick(ctx); err != nil {
		return fmt.Errorf("initial status write: %w", err)
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("sidecar stopping", "reason", ctx.Err())
			return nil
		case <-ticker.C:
			if err := r.tick(ctx); err != nil {
				return err
			}
		}
	}
}

func bootstrap(ctx context.Context, c *talos.Client, appID string) (*runner, error) {
	r := &runner{client: c, appID: appID, start: time.Now().UTC()}

	prev, found, err := c.LoadState(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("load hub state: %w", err)
	}
	r.prevStateFound = found
	if found && len(prev) > 0 {
		r.prevStateSnippet = clip(string(prev), 96)
	}

	state := []byte(time.Now().UTC().Format(time.RFC3339Nano))
	if err := c.SaveState(ctx, appID, state); err != nil {
		return nil, fmt.Errorf("save hub state: %w", err)
	}

	granted, msg, err := c.RequestPermission(ctx, appID, "net:internet",
		"Example Go app demonstrates the permission prompt (declared in manifest).")
	if err != nil {
		return nil, fmt.Errorf("request permission: %w", err)
	}
	r.netGranted = granted
	r.netMsg = msg

	if _, err := c.SendMessage(ctx, appID, tsExampleAppID, "app:example:from-go",
		[]byte(`{"from":"example-go","ts":`+fmt.Sprintf("%d", time.Now().Unix())+`}`)); err != nil {
		r.sendMsgNote = "SendMessage to " + tsExampleAppID + " failed (normal if that app is not running): " + err.Error()
	} else {
		r.sendMsgNote = "Routed a message to " + tsExampleAppID + " via the hub."
	}

	n, err := c.Broadcast(ctx, appID, "app:example:go-broadcast", []byte("hello from Go"))
	if err != nil {
		return nil, fmt.Errorf("broadcast: %w", err)
	}
	r.bcastN = n

	hb := fmt.Sprintf("bootstrap at %s\n", r.start.Format(time.RFC3339Nano))
	if err := c.WriteScopedFile(ctx, appID, "heartbeat.txt", []byte(hb)); err != nil {
		return nil, fmt.Errorf("write heartbeat: %w", err)
	}

	if err := c.Log(ctx, appID, "INFO", fmt.Sprintf("example-go-app ready; broadcast reached %d recipient(s)", n)); err != nil {
		slog.Warn("AppendPackageLog", "err", err)
	}

	return r, nil
}

func (r *runner) tick(ctx context.Context) error {
	r.ticks++

	raw, err := r.client.ReadScopedFile(ctx, r.appID, "heartbeat.txt")
	hbPreview := ""
	switch {
	case err != nil:
		hbPreview = "(could not read heartbeat.txt: " + err.Error() + ")"
	default:
		hbPreview = clip(strings.TrimSpace(string(raw)), 120)
	}

	st := statusPayload{
		SchemaVersion:       1,
		AppID:               r.appID,
		BootUTC:             r.start,
		LastTickUTC:         time.Now().UTC(),
		Ticks:               r.ticks,
		NetGranted:          r.netGranted,
		NetMessage:          r.netMsg,
		SendMessageNote:     r.sendMsgNote,
		BroadcastRecipients: r.bcastN,
		PrevStateFound:      r.prevStateFound,
		PrevStateSnippet:    r.prevStateSnippet,
		HeartbeatPreview:    hbPreview,
	}

	out, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	if err := r.client.WriteScopedFile(ctx, r.appID, statusFile, out); err != nil {
		return fmt.Errorf("write status file: %w", err)
	}

	if r.ticks%5 == 0 {
		if err := r.client.Log(ctx, r.appID, "DEBUG", fmt.Sprintf("example-go-app tick %d", r.ticks)); err != nil {
			slog.Warn("AppendPackageLog", "err", err)
		}
	}

	return nil
}

func clip(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
