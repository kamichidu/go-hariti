package yaml

import (
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type BuildScript struct {
	Windows string `yaml:"windows"`
	Mac     string `yaml:"mac"`
	Linux   string `yaml:"linux"`
	All     string `yaml:"all"`
}

type RemoteBundle struct {
	Name         string          `yaml:"name"`
	Aliases      []string        `yaml:"aliases"`
	Dependencies []*RemoteBundle `yaml:"dependencies"`
	EnableIfExpr string          `yaml:"enable-if"`
	BuildScript  *BuildScript    `yaml:"build"`
}

type LocalBundle struct {
	Path string `yaml:"path"`
}

type Bundle struct {
	Remote *RemoteBundle
	Local  *LocalBundle
}

type Bundles []Bundle

type BundlesFile struct {
	Version string  `yaml:"version"`
	Bundles Bundles `yaml:"bundles"`
}

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
			rb := &RemoteBundle{}
			buf, err := yaml.Marshal(item)
			if err != nil {
				return err
			}
			if err = yaml.Unmarshal(buf, rb); err != nil {
				return err
			}
			(*self)[i] = Bundle{Remote: rb}
		} else {
			lb := &LocalBundle{}
			buf, err := yaml.Marshal(item)
			if err != nil {
				return err
			}
			if err = yaml.Unmarshal(buf, lb); err != nil {
				return err
			}
			(*self)[i] = Bundle{Local: lb}
		}
	}
	return nil
}

func (self Bundle) MarshalYAML() (interface{}, error) {
	if self.Remote != nil {
		data := make(yaml.MapSlice, 0)
		data = append(data, yaml.MapItem{"name", self.Remote.Name})
		if len(self.Remote.Aliases) > 0 {
			data = append(data, yaml.MapItem{"aliases", self.Remote.Aliases})
		}
		if len(self.Remote.Dependencies) > 0 {
			data = append(data, yaml.MapItem{"dependencies", self.Remote.Dependencies})
		}
		if self.Remote.EnableIfExpr != "" {
			data = append(data, yaml.MapItem{"enable-if", self.Remote.EnableIfExpr})
		}
		if self.Remote.BuildScript != nil {
			bs := make(yaml.MapSlice, 0)
			if self.Remote.BuildScript.Windows != "" {
				bs = append(bs, yaml.MapItem{"windows", self.Remote.BuildScript.Windows})
			}
			if self.Remote.BuildScript.Mac != "" {
				bs = append(bs, yaml.MapItem{"mac", self.Remote.BuildScript.Mac})
			}
			if self.Remote.BuildScript.Linux != "" {
				bs = append(bs, yaml.MapItem{"linux", self.Remote.BuildScript.Linux})
			}
			if self.Remote.BuildScript.All != "" {
				bs = append(bs, yaml.MapItem{"all", self.Remote.BuildScript.All})
			}
			data = append(data, yaml.MapItem{"build", bs})
		}
		return data, nil
	} else if self.Local != nil {
		data := make(yaml.MapSlice, 0)
		data = append(data, yaml.MapItem{"path", self.Local.Path})
		return data, nil
	}
	return nil, nil
}

var _ yaml.Unmarshaler = (*Bundles)(nil)

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
