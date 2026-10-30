package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/0xzahed/ventox/internal/app"
	"github.com/0xzahed/ventox/internal/ui"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run is the testable entry point; it is the only place errors become exit codes.
func run(args []string) int {
	for _, a := range args {
		switch a {
		case "-h", "--help", "help":
			ui.PrintHelp()
			return 0
		case "-v", "--version", "version":
			ui.PrintVersion()
			return 0
		}
	}

	cfg, err := parseFlags(args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return 0
	case err != nil:
		ui.PrintError(fmt.Sprintf("Invalid arguments: %v", err))
		return 2
	}

	application := app.New(app.Deps{})
	if _, err := application.Lookup(cfg.Command); err != nil {
		ui.PrintError(err.Error())
		ui.PrintHelp()
		return 2
	}

	if app.RequiresRoot(cfg.Command) {
		if code, reexeced, err := ensureRoot(cfg.DryRun, args); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to acquire administrative privileges: %v", err))
			return 1
		} else if reexeced {
			return code
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch err := application.Run(ctx, cfg); {
	case err == nil:
		return 0
	case errors.Is(err, app.ErrCancelled):
		ui.PrintWarning("Operation cancelled by user. No changes were made.")
		return 0
	default:
		ui.PrintError(err.Error())
		return 1
	}
}

func parseFlags(args []string) (app.Config, error) {
	fs := flag.NewFlagSet("ventox", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = ui.PrintHelp

	var cfg app.Config
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "Simulate operation without writing")
	fs.BoolVar(&cfg.Verify, "verify", false, "Verify target data checksum after writing")
	fs.BoolVar(&cfg.BypassWin11, "bypass-win11", true, "Enable Windows 11 hardware bypasses")
	fs.StringVar(&cfg.DevicePath, "device", "", "Target USB device path")
	fs.StringVar(&cfg.ISOPath, "iso", "", "Source disk image / ISO path")
	fs.StringVar(&cfg.FS, "fs", "", "Target filesystem (exfat, fat32, ntfs, ext4, apfs)")
	fs.StringVar(&cfg.Label, "label", "", "Filesystem volume label")
	fs.BoolVar(&cfg.AutoConfirm, "yes", false, "Skip confirmation prompts")
	fs.BoolVar(&cfg.AutoConfirm, "y", false, "Skip confirmation prompts")
	fs.BoolVar(&cfg.JSONOutput, "json", false, "Structured JSON output")

	// The first non-flag argument is the subcommand.
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cfg.Command = strings.ToLower(args[0])
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return app.Config{}, err
	}
	return cfg, nil
}

// ensureRoot re-executes the binary under sudo when not privileged. It reports
// whether a re-exec happened together with the child's exit code.
func ensureRoot(dryRun bool, args []string) (code int, reexeced bool, err error) {
	if dryRun || runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return 0, false, nil
	}

	ui.Banner()
	ui.PrintInfo("Administrative privileges (sudo) required for direct block device access.")
	ui.PrintInfo("Re-executing with sudo...")
	fmt.Println()

	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	cmd := exec.Command("sudo", append([]string{exe}, args...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), true, nil
		}
		return 0, true, fmt.Errorf("failed to run sudo: %w", err)
	}
	return 0, true, nil
}
