package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"statusphere-client/internal/app"
	"statusphere-client/internal/auth"
)

var (
	uiMode       = flag.String("ui", "tui", "UI mode: tui, headless")
	registerFlag = flag.String("register", "", "Register with server: --register <server_url>")
	roomIDFlag   = flag.String("room", "", "Room ID for registration (omit to create new)")
)

func main() {
	flag.Parse()

	if *registerFlag != "" {
		if err := register(*registerFlag, *roomIDFlag); err != nil {
			fmt.Fprintf(os.Stderr, "registration failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := app.Run(ctx, *uiMode); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func register(serverURL, roomID string) error {
	cfg, err := auth.Register(serverURL, roomID)
	if err != nil {
		return err
	}
	fmt.Printf("Registered successfully!\n")
	fmt.Printf("  Config: %s\n", auth.ConfigPath())
	fmt.Printf("  Room:   %s\n", cfg.RoomID())
	fmt.Printf("  Device: %s\n", cfg.DeviceID())
	return nil
}
