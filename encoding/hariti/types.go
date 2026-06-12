package hariti

type Expr interface{}

type File struct {
	Bundles []Bundle
}

type Bundle interface {
}

type RemoteBundle struct {
	Uri          string
	Aliases      []string
	EnableIfExpr string
	Dependencies []string
	BuildScripts map[string][]string
}

type LocalBundle struct {
	Uri      string
	Includes []string
	Excludes []string
}

type BundleOptions []BundleOption

func (o BundleOptions) Apply(dest Bundle) {
	for _, option := range o {
		option.Apply(dest)
	}
	// apply defaults
	if b, ok := dest.(*RemoteBundle); ok {
		if b.Dependencies == nil {
			b.Dependencies = []string{}
		}
		if b.Aliases == nil {
			b.Aliases = []string{}
		}
		if b.BuildScripts == nil {
			b.BuildScripts = make(map[string][]string, 0)
		}
	}
}

type BundleOption interface {
	Apply(Bundle)
}

type UriOption struct {
	Value string
}

func (o *UriOption) Apply(dest Bundle) {
	if b, ok := dest.(*RemoteBundle); ok {
		b.Uri = o.Value
	}
}

type AliasesOption struct {
	Value []string
}

func (o *AliasesOption) Apply(dest Bundle) {
	if b, ok := dest.(*RemoteBundle); ok {
		b.Aliases = o.Value
	}
}

type EnableIfExprOption struct {
	Value string
}

func (o *EnableIfExprOption) Apply(dest Bundle) {
	if b, ok := dest.(*RemoteBundle); ok {
		b.EnableIfExpr = o.Value
	}
}

type DependenciesOption struct {
	Value []string
}

func (o *DependenciesOption) Apply(dest Bundle) {
	if b, ok := dest.(*RemoteBundle); ok {
		b.Dependencies = o.Value
	}
}

type BuildScriptsOption struct {
	Value map[string][]string
}

func (o *BuildScriptsOption) Apply(dest Bundle) {
	if b, ok := dest.(*RemoteBundle); ok {
		b.BuildScripts = o.Value
	}
}
