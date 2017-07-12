package hariti

import (
	"io"
	"io/ioutil"
	"net/url"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Bundle interface {
	GetName() string
	GetLocalPath() string
	GetAliases() []string
}

type BuildScript struct {
	Windows string `yaml:"windows"`
	Mac     string `yaml:"mac"`
	Linux   string `yaml:"linux"`
	All     string `yaml:"all"`
}

type RemoteBundle struct {
	// name
	Name string `yaml:"name"`
	// repository url
	URL *url.URL `yaml:"url"`
	// local directory path
	LocalPath string
	// aliases
	Aliases []string `yaml:"aliases"`
	// dependencies
	Dependencies []*RemoteBundle `yaml:"dependencies"`
	// vim expr
	EnableIfExpr string `yaml:"enable-if"`
	// build script
	BuildScript *BuildScript `yaml:"build"`
}

func createRemoteBundleFromMap(data map[string]interface{}) (*RemoteBundle, error) {
	bundle := &RemoteBundle{}
	buf, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bundle, yaml.Unmarshal(buf, bundle)
}

func (self *RemoteBundle) MarshalYAML() (interface{}, error) {
	data := make(yaml.MapSlice, 0)
	data = append(data, yaml.MapItem{"name", self.Name})
	if len(self.Aliases) > 0 {
		data = append(data, yaml.MapItem{"aliases", self.Aliases})
	}
	if len(self.Dependencies) > 0 {
		data = append(data, yaml.MapItem{"dependencies", self.Dependencies})
	}
	if self.EnableIfExpr != "" {
		data = append(data, yaml.MapItem{"enable-if", self.EnableIfExpr})
	}
	if self.BuildScript != nil {
		bs := make(yaml.MapSlice, 0)
		if self.BuildScript.Windows != "" {
			bs = append(bs, yaml.MapItem{"windows", self.BuildScript.Windows})
		}
		if self.BuildScript.Mac != "" {
			bs = append(bs, yaml.MapItem{"mac", self.BuildScript.Mac})
		}
		if self.BuildScript.Linux != "" {
			bs = append(bs, yaml.MapItem{"linux", self.BuildScript.Linux})
		}
		if self.BuildScript.All != "" {
			bs = append(bs, yaml.MapItem{"all", self.BuildScript.All})
		}
		data = append(data, yaml.MapItem{"build", bs})
	}
	return data, nil
}

func (self *RemoteBundle) GetName() string      { return self.Name }
func (self *RemoteBundle) GetLocalPath() string { return self.LocalPath }
func (self *RemoteBundle) GetAliases() []string { return self.Aliases }

var _ yaml.Marshaler = (*RemoteBundle)(nil)

type LocalBundle struct {
	// name
	Name string `yaml:"-"`
	// aliases
	Aliases []string `yaml:"-"`
	// path
	LocalPath string `yaml:"path"`
}

func createLocalBundleFromMap(data map[string]interface{}) (*LocalBundle, error) {
	bundle := &LocalBundle{}
	buf, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bundle, yaml.Unmarshal(buf, &bundle)
}

func (self *LocalBundle) GetName() string      { return filepath.Base(self.LocalPath) }
func (self *LocalBundle) GetLocalPath() string { return self.LocalPath }
func (self *LocalBundle) GetAliases() []string { return self.Aliases }

type Bundles []Bundle

func (self *Bundles) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var err error

	items := make([]map[string]interface{}, 0)
	if err = unmarshal(&items); err != nil {
		return err
	}
	*self = make(Bundles, len(items))
	for i, item := range items {
		var remoteFlag bool
		if _, ok := item["name"]; ok {
			remoteFlag = true
		} else if _, ok := item["path"]; ok {
			remoteFlag = false
		} else {
			continue
		}

		if remoteFlag {
			(*self)[i], err = createRemoteBundleFromMap(item)
		} else {
			(*self)[i], err = createLocalBundleFromMap(item)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

var _ yaml.Unmarshaler = (*Bundles)(nil)

type BundlesFile struct {
	Version string  `yaml:"version"`
	Bundles Bundles `yaml:"bundles"`
}

func MarshalBundles(w io.Writer, bundles Bundles) error {
	var err error

	_, err = w.Write([]byte("---\n"))
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(&BundlesFile{
		Version: "0.0",
		Bundles: bundles,
	})
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func UnmarshalBundles(r io.Reader) (Bundles, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	file := new(BundlesFile)
	if err = yaml.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	return file.Bundles, nil
}
