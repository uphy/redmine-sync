package sync

import (
	"errors"
	"io"
	"sort"
	"strconv"

	"github.com/mattn/go-redmine"
)

type (
	Sync struct {
		client     *redmine.Client
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

func (t *Names) FindIDByName(name string) (int, error) {
	if t.names == nil {
		names, err := t.init()
		if err != nil {
			return 0, err
		}
		t.names = names
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

func New(endpoint string, apiKey string) (*Sync, error) {
	client := redmine.NewClient(endpoint, apiKey)

	return &Sync{
		client:   client,
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
	}, nil
}

func (s *Sync) Import(reader io.Reader) (config *Config, changed bool, err error) {
	config, err = ReadConfig(reader)
	if err != nil {
		return nil, false, err
	}

	changed = false
	for _, project := range config.Projects {
		projectID := project.ID
		for _, ticket := range project.Tickets {
			ticketChanged, err := s.createTicket(projectID, 0, ticket)
			if err != nil {
				return nil, false, err
			}
			if ticketChanged {
				changed = true
			}
		}
	}
	return
}

func (s *Sync) Export(filter *redmine.IssueFilter, out io.Writer) (*Config, error) {
	var issues []redmine.Issue
	var err error
	if filter == nil {
		issues, err = s.client.Issues()
	} else {
		issues, err = s.client.IssuesByFilter(filter)
	}

	if err != nil {
		return nil, err
	}

	config := &Config{}
	tickets := map[int]*Ticket{}
	findOrCreateTicket := func(id int) *Ticket {
		if t, ok := tickets[id]; ok {
			return t
		}
		t := Ticket{ID: id}
		tickets[id] = &t
		return &t
	}
	for i := len(issues) - 1; i >= 0; i-- {
		src := issues[i]
		project := config.findOrCreateProject(src.Project.Id)
		dst := findOrCreateTicket(src.Id)
		s.mergeIssueToTicket(src, dst)
		if src.Parent != nil {
			parentID := src.Parent.Id
			parent := findOrCreateTicket(parentID)
			parent.Children = append(parent.Children, dst)
		} else {
			project.Tickets = append(project.Tickets, dst)
		}
	}
	for _, t := range tickets {
		sort.Slice(t.Children, func(i, j int) bool {
			return t.Children[i].ID < t.Children[j].ID
		})
	}
	return config, nil
}

func (s *Sync) createTicket(projectID int, parentTicketID int, ticket *Ticket) (bool, error) {
	var issue *redmine.Issue
	changed := false
	if ticket.ID == 0 {
		// create
		issue = &redmine.Issue{ProjectId: projectID}
		issue.ParentId = parentTicketID
		if err := s.mergeTicketToIssue(ticket, issue); err != nil {
			return changed, err
		}
		created, err := s.client.CreateIssue(*issue)
		if err != nil {
			return changed, err
		}
		issue = created
		// set created ticket ID in the input config file
		ticket.ID = issue.Id
		changed = true
	} else {
		// update
		updating, err := s.client.Issue(ticket.ID)
		if err != nil {
			return false, err
		}
		updating.ParentId = parentTicketID
		if err := s.mergeTicketToIssue(ticket, updating); err != nil {
			return changed, err
		}
		if err := s.client.UpdateIssue(*updating); err != nil {
			return changed, err
		}
		issue = updating
	}

	// create/update children
	for _, child := range ticket.Children {
		childChanged, err := s.createTicket(projectID, issue.Id, child)
		if err != nil {
			return false, err
		}
		if childChanged {
			changed = true
		}
	}
	return changed, nil
}

func (s *Sync) mergeIssueToTicket(src redmine.Issue, dst *Ticket) {
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

func (s *Sync) mergeTicketToIssue(src *Ticket, dst *redmine.Issue) error {
	dst.Id = src.ID
	if src.Subject != nil {
		dst.Subject = *src.Subject
	}
	if src.Description != nil {
		dst.Description = *src.Description
	}
	if src.Tracker != nil {
		trackerID, err := s.Trackers.FindIDByName(*src.Tracker)
		if err != nil {
			return err
		}
		dst.TrackerId = trackerID
	}
	if src.Status != nil {
		id, err := s.Statuses.FindIDByName(*src.Status)
		if err != nil {
			return err
		}
		dst.StatusId = id
	}
	if src.Priority != nil {
		priorityID, err := s.Priorities.FindIDByName(*src.Priority)
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
