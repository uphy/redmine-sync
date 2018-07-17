package importer

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Projects []*Project `yaml:"projects"`
}

type Project struct {
	ID      int       `yaml:"id"`
	Tickets []*Ticket `yaml:"tickets"`
}

// 変更可能な項目を定義。この構造体に含まれないフィールドについては更新されない。
type Ticket struct {
	ID          int     `yaml:"id"`
	Subject     *string `yaml:"subject"`
	Description *string `yaml:"description"`
	Tracker     *string `yaml:"tracker"`
	StartDate   *string `yaml:"start_date"`
	DueDate     *string `yaml:"due_date"`
	Priority    *string `yaml:"priority"`

	Children []*Ticket `yaml:"children,omitempty"`
}

func ReadConfig(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) Save(file string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, data, 0700)
}
