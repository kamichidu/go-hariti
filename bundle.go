package hariti

import (
	"net/url"
)

type Bundle struct {
	Name      string
	URL       *url.URL
	LocalPath string
}
