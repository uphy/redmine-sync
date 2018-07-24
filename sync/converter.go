package sync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gocarina/gocsv"
	redmine "github.com/uphy/go-redmine"
	yaml "gopkg.in/yaml.v2"
)

type (
	Converter struct {
		Trackers   *Names
		Priorities *Names
		Projects   *Names
		Statuses   *Names
	}
	Names struct {
		names []redmine.IdName
		init  func() ([]redmine.IdName, error)
	}
)

func (t *Names) initIfNeeded() error {
	if t.names == nil {
		names, err := t.init()
		if err != nil {
			return err
		}
		t.names = names
	}
	return nil
}

func (t *Names) FindNameByID(id int) (string, error) {
	if err := t.initIfNeeded(); err != nil {
		return "", err
	}
	for _, n := range t.names {
		if n.Id == id {
			return n.Name, nil
		}
	}
	return "", fmt.Errorf("no such id: %d", id)
}

func (t *Names) FindIDByName(name string) (int, error) {
	if err := t.initIfNeeded(); err != nil {
		return 0, err
	}
	for _, n := range t.names {
		if n.Name == name {
			return n.Id, nil
		}
	}
	id, err := strconv.Atoi(name)
	if err == nil {
		for _, n := range t.names {
			if n.Id == id {
				return n.Id, nil
			}
		}
	}
	return 0, errors.New("no such name: " + name)
}

func newConverter(client *redmine.Client) *Converter {
	return &Converter{
		Trackers: &Names{nil, client.Trackers},
		Priorities: &Names{nil, func() ([]redmine.IdName, error) {
			list, err := client.IssuePriorities()
			if err != nil {
				return nil, err
			}

			names := []redmine.IdName{}
			for _, item := range list {
				names = append(names, redmine.IdName{
					Id:   item.Id,
					Name: item.Name,
				})
			}
			return names, nil
		}},
		Projects: &Names{nil, func() ([]redmine.IdName, error) {
			list, err := client.Projects()
			if err != nil {
				return nil, err
			}

			names := []redmine.IdName{}
			for _, item := range list {
				names = append(names, redmine.IdName{
					Id:   item.Id,
					Name: item.Name,
				})
			}
			return names, nil
		}},
		Statuses: &Names{nil, func() ([]redmine.IdName, error) {
			list, err := client.IssueStatuses()
			if err != nil {
				return nil, err
			}

			names := []redmine.IdName{}
			for _, item := range list {
				names = append(names, redmine.IdName{
					Id:   item.Id,
					Name: item.Name,
				})
			}
			return names, nil
		}},
	}
}

func (c *Converter) Convert(issues []redmine.Issue) (*Config, error) {
	tickets := []*Ticket{}
	for _, issue := range issues {
		t := &Ticket{ID: issue.Id}
		c.mergeIssueToTicket(issue, t)
		tickets = append(tickets, t)
	}
	return c.toHierarchical(tickets)
}

func (c *Converter) mergeIssueToTicket(src redmine.Issue, dst *Ticket) {
	dst.Project = &src.Project.Name
	if src.Parent != nil {
		dst.ParentID = src.Parent.Id
	} else {
		dst.ParentID = 0
	}
	dst.ID = src.Id
	dst.Subject = &src.Subject
	dst.Description = &src.Description
	dst.Tracker = &src.Tracker.Name
	dst.Status = &src.Status.Name
	dst.Priority = &src.Priority.Name
	if src.StartDate != "" {
		dst.StartDate = &src.StartDate
	}
	if src.DueDate != "" {
		dst.DueDate = &src.DueDate
	}
}

func (c *Converter) mergeTicketToIssue(src *Ticket, dst *redmine.Issue) error {
	dst.Id = src.ID
	if src.Project != nil {
		id, err := c.Projects.FindIDByName(*src.Project)
		if err != nil {
			return err
		}
		dst.ProjectId = id
	}
	dst.ParentId = src.ParentID
	if src.ParentID == 0 {
		dst.Parent = nil
	} else {
		dst.Parent = &redmine.Id{Id: src.ParentID}
	}
	if src.Subject != nil {
		dst.Subject = *src.Subject
	}
	if src.Description != nil {
		dst.Description = *src.Description
	}
	if src.Tracker != nil {
		trackerID, err := c.Trackers.FindIDByName(*src.Tracker)
		if err != nil {
			return err
		}
		dst.TrackerId = trackerID
	}
	if src.Status != nil {
		id, err := c.Statuses.FindIDByName(*src.Status)
		if err != nil {
			return err
		}
		dst.StatusId = id
	}
	if src.Priority != nil {
		priorityID, err := c.Priorities.FindIDByName(*src.Priority)
		if err != nil {
			return err
		}
		dst.PriorityId = priorityID
	}
	if src.StartDate != nil {
		dst.StartDate = *src.StartDate
	}
	if src.DueDate != nil {
		dst.DueDate = *src.DueDate
	}
	return nil
}

func (c *Converter) ReadConfig(file *os.File) (*Config, error) {
	ext := c.extension(file.Name())
	switch ext {
	case ".yaml", ".yml":
		return c.readConfigYAML(file)
	case ".csv", "":
		return c.readConfigCSV(file)
	default:
		return nil, errors.New("unsupported extension: " + ext)
	}
}

func (c *Converter) SaveConfig(file *os.File, config *Config) error {
	ext := c.extension(file.Name())
	switch ext {
	case ".yaml", ".yml":
		return c.SaveConfigYAML(file, config)
	case ".csv", "":
		return c.SaveConfigCSV(file, config)
	default:
		return errors.New("unsupported extension: " + ext)
	}
}

func (c *Converter) extension(name string) string {
	return strings.ToLower(filepath.Ext(name))
}

func (c *Converter) readConfigCSV(reader io.Reader) (*Config, error) {
	var csvTickets []*Ticket
	if err := gocsv.Unmarshal(reader, &csvTickets); err != nil {
		return nil, err
	}
	return c.toHierarchical(csvTickets)
}

func (c *Converter) readConfigYAML(reader io.Reader) (*Config, error) {
	var config Config
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	if _, err := c.toFlat(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Converter) SaveConfigYAML(writer io.Writer, config *Config) error {
	encoder := yaml.NewEncoder(writer)
	return encoder.Encode(config)
}

func (c *Converter) SaveConfigCSV(writer io.Writer, config *Config) error {
	w := gocsv.DefaultCSVWriter(writer)
	tickets, err := c.toFlat(config)
	if err != nil {
		return err
	}
	for i, t := range tickets {
		if i == 0 {
			if err := gocsv.MarshalCSV([]*Ticket{t}, w); err != nil {
				return err
			}
		} else {
			if err := gocsv.MarshalCSVWithoutHeaders([]*Ticket{t}, w); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Converter) toFlat(config *Config) ([]*Ticket, error) {
	tickets := []*Ticket{}
	for _, p := range config.Projects {
		projectName, err := c.Projects.FindNameByID(p.ID)
		if err != nil {
			return nil, err
		}
		for _, ticket := range p.Tickets {
			ticket.ParentID = 0
			tickets = c.collectTickets(tickets, ticket, projectName)
		}
	}
	sort.Slice(tickets, func(i, j int) bool {
		p1 := *tickets[i].Project
		p2 := *tickets[j].Project
		c := strings.Compare(p1, p2)
		switch c {
		case 1:
			return false
		case -1:
			return true
		}
		return tickets[i].ID < tickets[j].ID
	})
	return tickets, nil
}

func (c *Converter) collectTickets(tickets []*Ticket, t *Ticket, projectName string) []*Ticket {
	t.Project = &projectName
	tickets = append(tickets, t)
	if t.Children != nil {
		for _, child := range t.Children {
			child.ParentID = t.ID
			tickets = c.collectTickets(tickets, child, projectName)
		}
	}
	return tickets
}

func (c *Converter) toHierarchical(tickets []*Ticket) (*Config, error) {
	idToTickets := map[int]*Ticket{}
	for _, t := range tickets {
		idToTickets[t.ID] = t
	}

	config := &Config{}
	findOrCreateTicket := func(id int) *Ticket {
		if t, ok := idToTickets[id]; ok {
			return t
		}
		t := Ticket{ID: id}
		idToTickets[id] = &t
		return &t
	}
	for _, t := range tickets {
		if t.ParentID == 0 {
			projectID, err := c.Projects.FindIDByName(*t.Project)
			if err != nil {
				return nil, err
			}
			project := config.findOrCreateProject(projectID)
			project.Tickets = append(project.Tickets, t)
		} else {
			parent := findOrCreateTicket(t.ParentID)
			parent.Children = append(parent.Children, t)
		}
	}
	sortTickets := func(tickets []*Ticket) {
		sort.Slice(tickets, func(i, j int) bool {
			return tickets[i].ID < tickets[j].ID
		})
	}

	for _, t := range tickets {
		sortTickets(t.Children)
	}
	for _, p := range config.Projects {
		sortTickets(p.Tickets)
	}
	sort.Slice(config.Projects, func(i, j int) bool {
		return config.Projects[i].ID < config.Projects[j].ID
	})
	return config, nil
}
