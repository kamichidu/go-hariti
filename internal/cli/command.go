package cli

type Command interface {
	Name() string
	Run(ctx *Context, args []string) error
}

var registry []Command

func Register(cmd Command) {
	registry = append(registry, cmd)
}

func All() []Command {
	return registry
}
