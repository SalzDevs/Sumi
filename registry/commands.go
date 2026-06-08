package registry

import (
	"fmt"
	"sort"

	"sumi/editor"
)

// Command is a registered editor command.
type Command struct {
	Name        string
	Description string
	Handler     func(e *editor.Editor, args []string) error
	MinArgs     int // minimum required arguments
	MaxArgs     int // maximum allowed arguments; -1 means unlimited
}

// CommandRegistry holds all available commands.
type CommandRegistry struct {
	commands map[string]*Command
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Command),
	}
}

// Register adds a command to the registry.
func (r *CommandRegistry) Register(name, description string, minArgs, maxArgs int, handler func(*editor.Editor, []string) error) {
	r.commands[name] = &Command{
		Name:        name,
		Description: description,
		Handler:     handler,
		MinArgs:     minArgs,
		MaxArgs:     maxArgs,
	}
}

// Execute looks up a command by name and runs it with the given arguments.
func (r *CommandRegistry) Execute(e *editor.Editor, name string, args []string) error {
	cmd, ok := r.commands[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}

	argCount := len(args)
	if argCount < cmd.MinArgs {
		return fmt.Errorf("%s: requires at least %d argument(s)", name, cmd.MinArgs)
	}
	if cmd.MaxArgs >= 0 && argCount > cmd.MaxArgs {
		return fmt.Errorf("%s: takes at most %d argument(s)", name, cmd.MaxArgs)
	}

	return cmd.Handler(e, args)
}

// Names returns all registered command names, sorted.
func (r *CommandRegistry) Names() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
