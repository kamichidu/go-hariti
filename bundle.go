package hariti

type Bundle interface {
	GetName() string
	GetLocalPath() string
	GetAliases() []string
}
