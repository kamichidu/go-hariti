package hariti

import (
	"net/url"
)

type Bundle interface {
	GetName() string
	GetLocalPath() string
	GetAliases() []string
	GetDependencies() []Bundle
}

type RemoteBundle struct {
	// name
	Name string
	// repository url
	URL *url.URL
	// local directory path
	LocalPath string
	// aliases
	Aliases []string
	// dependencies
	Dependencies []Bundle
}

func (self *RemoteBundle) GetName() string           { return self.Name }
func (self *RemoteBundle) GetLocalPath() string      { return self.LocalPath }
func (self *RemoteBundle) GetAliases() []string      { return self.Aliases }
func (self *RemoteBundle) GetDependencies() []Bundle { return self.Dependencies }

type LocalBundle struct {
	Name         string
	LocalPath    string
	Aliases      []string
	Dependencies []Bundle
}

func (self *LocalBundle) GetName() string           { return self.Name }
func (self *LocalBundle) GetLocalPath() string      { return self.LocalPath }
func (self *LocalBundle) GetAliases() []string      { return self.Aliases }
func (self *LocalBundle) GetDependencies() []Bundle { return self.Dependencies }
