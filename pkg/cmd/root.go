package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/internal/ui"
)

type command struct {
	name    string
	aliases []string
	run     func(args []string) error
}

// Root wires all CLI commands together.
type Root struct {
	version  string
	commands map[string]*command
}

// New creates a Root command registry.
func New(version string) *Root {
	r := &Root{
		version:  version,
		commands: make(map[string]*command),
	}

	r.register(&command{name: "config", run: r.cmdConfig})
	r.register(&command{name: "theme", aliases: []string{"themes"}, run: r.cmdTheme})
	r.register(&command{name: "login", run: r.cmdLogin})
	r.register(&command{name: "sync", run: r.cmdSync})
	r.register(&command{name: "digest", run: r.cmdDigest})
	r.register(&command{name: "add", run: r.cmdAdd})
	r.register(&command{name: "list", aliases: []string{"ls"}, run: r.cmdList})
	r.register(&command{name: "open", run: r.cmdOpen})
	r.register(&command{name: "save", run: r.cmdSave})
	r.register(&command{name: "mute", run: r.cmdMute})
	r.register(&command{name: "boost", run: r.cmdBoost})
	r.register(&command{name: "rules", run: r.cmdRules})
	r.register(&command{name: "sources", run: r.cmdSources})
	r.register(&command{name: "curate", run: r.cmdCurate})
	r.register(&command{name: "stream", run: r.cmdStream})
	r.register(&command{name: "daemon", run: r.cmdDaemon})
	r.register(&command{name: "serve", run: r.cmdServe})
	r.register(&command{name: "setup", run: r.cmdSetup})
	r.register(&command{name: "help", aliases: []string{"-h", "--help"}, run: r.cmdHelp})
	r.register(&command{name: "version", aliases: []string{"-v", "--version"}, run: r.cmdVersion})
	r.register(&command{name: "sync-and-run", run: r.cmdSyncAndRun})

	return r
}

func (r *Root) register(cmd *command) {
	r.commands[cmd.name] = cmd
	for _, alias := range cmd.aliases {
		r.commands[alias] = cmd
	}
}

// Execute dispatches to the appropriate subcommand.
func (r *Root) Execute(args []string) error {
	if len(args) == 0 {
		return r.runApp()
	}

	if cmd, ok := r.commands[args[0]]; ok {
		if err := cmd.run(args[1:]); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("unknown command: %s", args[0])
}

func (r *Root) runApp() error {
	if isFirstRun() {
		if err := runFirstTimeSetup(); err != nil {
			return err
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		// allow running without a store
		st = nil
	}
	if st != nil {
		defer st.Close()
	}

	model := ui.NewModel(cfg, st)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func (r *Root) cmdSyncAndRun(args []string) error {
	if err := r.cmdSync(nil); err != nil {
		fmt.Println("⚠️  Local sync failed; continuing with cached items.")
	}
	return r.runApp()
}

func withStore(fn func(*store.Store) error) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()
	return fn(st)
}
