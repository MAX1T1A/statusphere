package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"statusphere-client/internal/app"
	"statusphere-client/internal/auth"
)

var (
	uiMode = flag.String("ui", "tui", "UI mode: tui, headless")

	screenshotFlag = flag.Bool("screenshot", false, "Render a room member's card offscreen as ANSI to stdout")
	ssDeviceFlag   = flag.String("ss-device", "", "Target device id for --screenshot (default: a member who is playing)")
	ssModeFlag     = flag.String("ss-mode", "music", "Screenshot modal: music or screen")
	ssWidthFlag    = flag.Int("ss-width", 100, "Screenshot width")
	ssHeightFlag   = flag.Int("ss-height", 32, "Screenshot height")
	ssWaitFlag     = flag.Int("ss-wait", 5, "Seconds to collect the room feed before rendering")

	registerFlag  = flag.String("register", "", "Create a new account on <server_url>")
	linkFlag      = flag.String("link", "", "Link this device to an existing account on <server_url> (needs --code)")
	codeFlag      = flag.String("code", "", "Link code produced by --new-device")
	recoverFlag   = flag.String("recover", "", "Recover an account on <server_url> (needs --account and --secret)")
	accountFlag   = flag.String("account", "", "Account id for --recover")
	secretFlag    = flag.String("secret", "", "Account secret for --recover")
	newDeviceFlag = flag.Bool("new-device", false, "Print a link code to add another device to this account")
	inviteFlag    = flag.Bool("invite", false, "Print an invite code for your room")
	joinFlag      = flag.String("join", "", "Join a room using an invite <code>")
	devicesFlag   = flag.Bool("devices", false, "List devices on this account")
	revokeFlag    = flag.String("revoke", "", "Revoke a device by <device_id>")
	membersFlag   = flag.Bool("members", false, "List members of your room")
	kickFlag      = flag.String("kick", "", "Remove a member by <account_id>")
	setNameFlag   = flag.String("set-name", "", "Set your account's display name")
)

func main() {
	flag.Parse()

	if err := dispatch(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func dispatch() error {
	switch {
	case *screenshotFlag:
		return runScreenshot()
	case *registerFlag != "":
		return register(*registerFlag)
	case *linkFlag != "":
		return linkDevice(*linkFlag, *codeFlag)
	case *recoverFlag != "":
		return recoverAccount(*recoverFlag, *accountFlag, *secretFlag)
	case *newDeviceFlag:
		return withConfig(func(c *auth.Config) error {
			code, err := c.NewDeviceCode()
			if err != nil {
				return err
			}
			fmt.Printf("Run on the new device:\n  statusphere --link %s --code %s\n", c.ServerURL, code)
			return nil
		})
	case *inviteFlag:
		return withConfig(func(c *auth.Config) error {
			code, err := c.Invite()
			if err != nil {
				return err
			}
			fmt.Printf("Share with a friend:\n  statusphere --join %s\n", auth.EncodeInvite(c.ServerURL, code))
			return nil
		})
	case *joinFlag != "":
		return joinRoom(*joinFlag)
	case *devicesFlag:
		return withConfig(listDevices)
	case *revokeFlag != "":
		return withConfig(func(c *auth.Config) error {
			ok, err := c.Revoke(*revokeFlag)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("no such device on your account: %s", *revokeFlag)
			}
			fmt.Printf("Revoked device %s\n", *revokeFlag)
			return nil
		})
	case *setNameFlag != "":
		return withConfig(func(c *auth.Config) error {
			if err := c.SetAccountName(*setNameFlag); err != nil {
				return err
			}
			fmt.Printf("Account name set to %q (visible to your room on reconnect)\n", *setNameFlag)
			return nil
		})
	case *membersFlag:
		return withConfig(listMembers)
	case *kickFlag != "":
		return withConfig(func(c *auth.Config) error {
			ok, err := c.Kick(*kickFlag)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("not a removable member of your room: %s", *kickFlag)
			}
			fmt.Printf("Removed %s\n", *kickFlag)
			return nil
		})
	default:
		return run()
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	return app.Run(ctx, *uiMode)
}

func runScreenshot() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	out, err := app.Screenshot(ctx, app.ScreenshotOpts{
		Device: *ssDeviceFlag,
		Mode:   *ssModeFlag,
		Width:  *ssWidthFlag,
		Height: *ssHeightFlag,
		Wait:   time.Duration(*ssWaitFlag) * time.Second,
	})
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

func joinRoom(arg string) error {
	server, code := auth.DecodeInvite(arg)

	cfg, err := auth.Load()
	needRegister := err != nil || (server != "" && cfg.ServerURL != server)
	if needRegister {
		if server == "" {
			return fmt.Errorf("no account found; register first: statusphere --register <server_url>")
		}
		cfg, err = auth.Register(server)
		if err != nil {
			return err
		}
		fmt.Printf("Account created on %s\n", server)
	}

	if err := cfg.Join(code); err != nil {
		return err
	}
	fmt.Printf("Joined room %s\n", cfg.RoomID)
	return nil
}

func withConfig(fn func(*auth.Config) error) error {
	cfg, err := auth.Load()
	if err != nil {
		return fmt.Errorf("no account found; register first: statusphere --register <server_url>")
	}
	return fn(cfg)
}

func register(serverURL string) error {
	cfg, err := auth.Register(serverURL)
	if err != nil {
		return err
	}
	fmt.Printf("Account created.\n")
	fmt.Printf("  Config:  %s\n", auth.ConfigPath())
	fmt.Printf("  Account: %s\n", cfg.AccountID)
	fmt.Printf("  Room:    %s\n", cfg.RoomID)
	fmt.Printf("  Device:  %s\n", cfg.DeviceID)
	fmt.Printf("\nInvite friends:      statusphere --invite\n")
	fmt.Printf("Add another device:  statusphere --new-device\n")
	return nil
}

func recoverAccount(serverURL, accountID, secret string) error {
	if accountID == "" || secret == "" {
		return fmt.Errorf("--recover requires --account <account_id> and --secret <account_secret>")
	}
	cfg, err := auth.Recover(serverURL, accountID, secret)
	if err != nil {
		return err
	}
	fmt.Printf("Account recovered on a new device.\n")
	fmt.Printf("  Account: %s\n", cfg.AccountID)
	fmt.Printf("  Room:    %s\n", cfg.RoomID)
	fmt.Printf("  Device:  %s\n", cfg.DeviceID)
	return nil
}

func linkDevice(serverURL, code string) error {
	if code == "" {
		return fmt.Errorf("--link requires --code <code> (get one with --new-device on an existing device)")
	}
	cfg, err := auth.LinkDevice(serverURL, code)
	if err != nil {
		return err
	}
	fmt.Printf("Device linked.\n")
	fmt.Printf("  Account: %s\n", cfg.AccountID)
	fmt.Printf("  Room:    %s\n", cfg.RoomID)
	fmt.Printf("  Device:  %s\n", cfg.DeviceID)
	return nil
}

func listDevices(c *auth.Config) error {
	devices, err := c.Devices()
	if err != nil {
		return err
	}
	for _, d := range devices {
		marker := ""
		if d.DeviceID == c.DeviceID {
			marker = " (this device)"
		}
		state := "active"
		if d.Revoked {
			state = "revoked"
		}
		name := d.Name
		if name == "" {
			name = "—"
		}
		fmt.Printf("%s  %-16s  %s%s\n", d.DeviceID, name, state, marker)
	}
	return nil
}

func listMembers(c *auth.Config) error {
	members, err := c.Members()
	if err != nil {
		return err
	}
	for _, m := range members {
		marker := ""
		if m.AccountID == c.AccountID {
			marker = " (you)"
		}
		name := m.Name
		if name == "" {
			name = "—"
		}
		fmt.Printf("%s  %-20s %s%s\n", m.AccountID, name, m.Role, marker)
	}
	return nil
}
