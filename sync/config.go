package sync

import (
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type (
	Config struct {
		Projects []*Project `yaml:"projects"`
	}

	Project struct {
		ID      int       `yaml:"id"`
		Tickets []*Ticket `yaml:"tickets"`
	}

	// 変更可能な項目を定義。この構造体に含まれないフィールドについては更新されない。
	Ticket struct {
		ID          int     `yaml:"id"`
		Subject     *string `yaml:"subject"`
		Status      *string `yaml:"status"`
		Description *string `yaml:"description"`
		Tracker     *string `yaml:"tracker"`
		StartDate   *string `yaml:"start_date"`
		DueDate     *string `yaml:"due_date"`
		Priority    *string `yaml:"priority"`

		Children []*Ticket `yaml:"children,omitempty"`
	}
)

func ReadConfigFile(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadConfig(f)
}

func ReadConfig(reader io.Reader) (*Config, error) {
	var c Config
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) SaveFile(file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Save(f)
}

func (c *Config) Save(writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	if err := encoder.Encode(c); err != nil {
		return err
	}
	return nil
}

func (c *Config) findOrCreateProject(id int) *Project {
	for _, p := range c.Projects {
		if p.ID == id {
			return p
		}
	}
	p := &Project{ID: id}
	c.Projects = append(c.Projects, p)
	return p
}
