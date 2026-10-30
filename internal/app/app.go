// Package app is the application Facade: it wires device discovery, image
// inspection, planning and execution behind a small set of CLI commands.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/0xzahed/ventox/internal/device"
	"github.com/0xzahed/ventox/internal/engine"
)

// ErrCancelled signals that the user declined to proceed; it is not a failure.
var ErrCancelled = errors.New("operation cancelled by user")

// ErrUnknownCommand is returned for subcommands that are not registered.
var ErrUnknownCommand = errors.New("unknown command")

// Config holds the parsed CLI options.
type Config struct {
	Command     string
	DryRun      bool
	Verify      bool
	BypassWin11 bool
	DevicePath  string
	ISOPath     string
	FS          string
	Label       string
	AutoConfirm bool
	JSONOutput  bool
}

// Command is the Command pattern interface: one implementation per CLI subcommand.
type Command interface {
	Run(ctx context.Context, cfg Config) error
}

// Deps are the collaborators injected into commands (Dependency Inversion).
type Deps struct {
	Devices     device.DeviceProvider
	NewExecutor func() *engine.Executor
	Stdout      io.Writer
}

// App dispatches configs to registered commands.
type App struct {
	commands map[string]Command
	fallback Command
}

// New builds an App with the default commands, filling unset dependencies with production defaults.
func New(deps Deps) *App {
	if deps.Devices == nil {
		deps.Devices = device.DefaultProvider
	}
	if deps.NewExecutor == nil {
		deps.NewExecutor = func() *engine.Executor { return engine.NewExecutor() }
	}
	if deps.Stdout == nil {
		deps.Stdout = os.Stdout
	}

	list := listCommand{deps: deps}
	format := formatCommand{deps: deps}
	return &App{
		commands: map[string]Command{
			"list":   list,
			"ls":     list,
			"format": format,
			"wipe":   format,
		},
		fallback: bootCommand{deps: deps},
	}
}

// Lookup resolves the command for a name; an empty name selects the boot creator.
func (a *App) Lookup(name string) (Command, error) {
	name = strings.ToLower(name)
	if name == "" {
		return a.fallback, nil
	}
	if c, ok := a.commands[name]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("%w %q", ErrUnknownCommand, name)
}

// RequiresRoot reports whether the named command writes to block devices.
func RequiresRoot(command string) bool {
	switch strings.ToLower(command) {
	case "list", "ls":
		return false
	}
	return true
}

// Run executes the configured command.
func (a *App) Run(ctx context.Context, cfg Config) error {
	cmd, err := a.Lookup(cfg.Command)
	if err != nil {
		return err
	}
	return cmd.Run(ctx, cfg)
}
