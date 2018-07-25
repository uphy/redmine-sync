package sync

type (
	Config struct {
		Projects []*Project `yaml:"projects"`
	}

	Project struct {
		ID      int       `yaml:"id"`
		Tickets []*Ticket `yaml:"tickets"`
	}

	// 変更可能な項目を定義。この構造体に含まれないフィールドについては更新されない。
	// []Issue => Config => []Ticket => Redmine
	// YAML =ReadConfigYAML=> Config => []Ticket => Redmine
	// CSV =ReadConfigCSV=> Config
	Ticket struct {
		Project     *string `yaml:"-" csv:"Project"`
		ID          int     `yaml:"id" csv:"ID"`
		ParentID    int     `yaml:"-" csv:"Parent ID"`
		Subject     *string `yaml:"subject" csv:"Subject"`
		Assignee    *string `yaml:"assignee" csv:"Assignee"`
		Status      *string `yaml:"status" csv:"Status"`
		Description *string `yaml:"description" csv:"Description"`
		Tracker     *string `yaml:"tracker" csv:"Tracker"`
		StartDate   *string `yaml:"start_date" csv:"Start Date"`
		DueDate     *string `yaml:"due_date" csv:"Due Date"`
		Priority    *string `yaml:"priority" csv:"Priority"`

		Children []*Ticket `yaml:"children,omitempty" csv:"-"`
	}
)

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
