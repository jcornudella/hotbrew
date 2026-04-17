package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/sources/xbookmarks"
)

func (r *Root) cmdXAuth(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var configured string
	if cfg.X != nil {
		configured = cfg.X.ClientID
	}
	clientID := xbookmarks.ClientID(configured)
	if clientID == "" {
		return fmt.Errorf("x client id not set: add `x:\n  client_id: <id>` to hotbrew.yaml, or export HOTBREW_X_CLIENT_ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	tok, err := xbookmarks.RunPKCEFlow(ctx, clientID)
	if err != nil {
		return fmt.Errorf("authorize: %w", err)
	}

	fmt.Printf("\n✅ Authorized. Token saved to %s\n", xbookmarks.TokenPath())
	fmt.Printf("   Expires: %s\n", tok.ExpiresAt.Local().Format(time.RFC1123))
	fmt.Println("   Run `hotbrew bookmarks` to pull your feed.")
	return nil
}
