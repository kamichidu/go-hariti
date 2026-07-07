package cli

import (
	"github.com/kamichidu/go-flagshim"
)

var registry []flagshim.Command

func Register(cmd flagshim.Command) {
	registry = append(registry, cmd)
}

func All() []flagshim.Command {
	return registry
}
