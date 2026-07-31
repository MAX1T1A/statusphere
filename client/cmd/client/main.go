package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"statusphere-client/internal/app"
	"statusphere-client/internal/auth"
	"statusphere-client/internal/presence"
	"statusphere-client/internal/privacy"
)

var (
	uiMode      = flag.String("ui", "tui", "UI mode: tui, headless, json (roster as JSON lines on stdout, for external UIs)")
	intervalArg = flag.Duration("interval", 2*time.Second, "How often to collect and publish (a headless box wants seconds, not milliseconds)")
	setKindFlag = flag.String("set-kind", "", "What this machine is: desktop or server (server cards are read for metrics, not for open windows)")

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
	postPhotoFlag = flag.String("post-photo", "", "Share <path> as your current photo status, replacing any previous one")

	incognitoFlag = flag.String("incognito", "", "Hide what you're doing from the room: on, off, status, or a duration like 45m")
	noteFlag      = flag.String("note", "", "Short note the room sees next to your hidden badge (empty clears it)")
	publishedFlag = flag.Bool("published", false, "Print the snapshot this device would send right now, after the privacy filter")
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
	case *incognitoFlag != "":
		return runIncognito(*incognitoFlag)
	case isSet("note"):
		return runNote(*noteFlag)
	case *publishedFlag:
		return runPublished()
	case isSet("set-kind"):
		return runSetKind(*setKindFlag)
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
	case *postPhotoFlag != "":
		return withConfig(func(c *auth.Config) error {
			info, err := c.PostPhoto(*postPhotoFlag)
			if err != nil {
				return err
			}
			fmt.Printf("Shared. Visible to your room until %s\n", info.ExpiresAt)
			return nil
		})
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
	// The roster drops a device it has not heard from in five minutes, so an
	// interval anywhere near that makes the card blink offline between ticks.
	if *intervalArg < time.Second || *intervalArg > 2*time.Minute {
		return fmt.Errorf("--interval takes 1s to 2m, got %s", *intervalArg)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	return app.Run(ctx, app.Options{UI: *uiMode, Interval: *intervalArg})
}

func runSetKind(kind string) error {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case presence.KindDesktop, presence.KindServer:
	case "":
		kind = presence.KindDesktop
	default:
		return fmt.Errorf("--set-kind takes %s or %s", presence.KindDesktop, presence.KindServer)
	}

	return withConfig(func(c *auth.Config) error {
		c.Kind = kind
		if kind == presence.KindDesktop {
			c.Kind = ""
		}
		if err := c.Save(); err != nil {
			return err
		}
		fmt.Printf("This machine now reports itself as a %s (restart the running client to publish it)\n", kind)
		return nil
	})
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

func runIncognito(arg string) error {
	if isSet("note") {
		if _, err := privacy.SetNote(strings.TrimSpace(*noteFlag)); err != nil {
			return err
		}
	}

	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "status":
		p, err := privacy.Load()
		if err != nil {
			return err
		}
		fmt.Print(incognitoStatus(p))
		return nil
	case "on":
		return applyIncognito(privacy.ModeIncognito, 0)
	case "off":
		return applyIncognito(privacy.ModeNormal, 0)
	}

	d, err := time.ParseDuration(arg)
	if err != nil || d <= 0 {
		return fmt.Errorf("--incognito takes on, off, status, or a duration like 45m")
	}
	return applyIncognito(privacy.ModeIncognito, d)
}

func applyIncognito(mode string, d time.Duration) error {
	p, err := privacy.Set(mode, d)
	if err != nil {
		return err
	}
	fmt.Print(incognitoStatus(p))
	return nil
}

func incognitoStatus(p privacy.Policy) string {
	var b strings.Builder

	if p.Hidden() {
		b.WriteString("Hidden")
		if until, ok := p.Expires(); ok {
			fmt.Fprintf(&b, " until %s (%s left)", until.Format("15:04"), timeLeft(time.Until(until)))
		}
		if p.Note != "" {
			fmt.Fprintf(&b, " · note %q", p.Note)
		}
		b.WriteString("\n")
	} else {
		b.WriteString("Visible\n")
	}

	prof := p.Active()
	fmt.Fprintf(&b, "Room sees:  apps %s · music %s · system %s · custom %s\n", prof.Apps, prof.Music, prof.System, prof.Custom)
	if p.Hidden() && !p.Announce {
		b.WriteString("Announce off: the room sees a quiet card, with no reason given.\n")
	}
	fmt.Fprintf(&b, "Full check: statusphere --published\nSettings:   %s\n", privacy.Path())
	return b.String()
}

func timeLeft(d time.Duration) string {
	d = d.Round(time.Minute)
	if d >= time.Hour {
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func runNote(note string) error {
	p, err := privacy.SetNote(strings.TrimSpace(note))
	if err != nil {
		return err
	}
	if p.Note == "" {
		fmt.Println("Note cleared")
		return nil
	}
	fmt.Printf("Note set to %q, shown while you're hidden\n", p.Note)
	return nil
}

func runPublished() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := app.Published(ctx)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func isSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
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
