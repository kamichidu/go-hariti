package hariti

type BundleType uint

const (
	BundleTypeGit BundleType = iota
)

type Bundle struct {
	Id        string
	Type      BundleType
	Url       string
	LocalPath string
}
