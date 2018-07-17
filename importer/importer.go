package importer

import (
	"errors"
	"fmt"
	"io"

	"github.com/mattn/go-redmine"
)

type (
	Importer struct {
		client     *redmine.Client
		trackers   *names
		priorities *names
	}
	names struct {
		names []redmine.IdName
	}
)

func (t *names) findIDByName(name string) (int, error) {
	for _, n := range t.names {
		if n.Name == name {
			return n.Id, nil
		}
	}
	return 0, errors.New("no such name: " + name)
}

func NewImporter(endpoint string, apiKey string) (*Importer, error) {
	client := redmine.NewClient(endpoint, apiKey)
	trackers, err := client.Trackers()
	if err != nil {
		return nil, err
	}

	p, err := client.IssuePriorities()
	if err != nil {
		return nil, err
	}
	priorities := []redmine.IdName{}
	for _, priority := range p {
		priorities = append(priorities, redmine.IdName{
			Id:   priority.Id,
			Name: priority.Name,
		})
	}

	return &Importer{
		client:     client,
		trackers:   &names{trackers},
		priorities: &names{priorities},
	}, nil
}

func (i *Importer) Import(file string) error {
	config, err := ReadConfig(file)
	if err != nil {
		return err
	}

	changed := false
	defer func() {
		if changed {
			if err := config.Save(file); err != nil {
				fmt.Println(err)
			}
		}
	}()
	for _, project := range config.Projects {
		projectID := project.ID
		for _, ticket := range project.Tickets {
			ticketChanged, err := i.createTicket(projectID, 0, ticket)
			if err != nil {
				return err
			}
			if ticketChanged {
				changed = true
			}
		}
	}
	return nil
}

func (i *Importer) Export(out io.Writer) error {
	issues, err := i.client.Issues()
	if err != nil {
		return err
	}
	for _, issue := range issues {
		fmt.Println(issue.Id)
	}
	return nil
}

func (i *Importer) createTicket(projectID int, parentTicketID int, ticket *Ticket) (bool, error) {
	var issue *redmine.Issue
	changed := false
	if ticket.ID == 0 {
		// create
		issue = &redmine.Issue{ProjectId: projectID}
		if err := i.mergeTickets(ticket, issue); err != nil {
			return changed, err
		}
		created, err := i.client.CreateIssue(*issue)
		if err != nil {
			return changed, err
		}
		issue = created
		ticket.ID = issue.Id
		changed = true
	} else {
		// update
		updating, err := i.client.Issue(ticket.ID)
		if err != nil {
			return false, err
		}
		if err := i.mergeTickets(ticket, updating); err != nil {
			return changed, err
		}
		if err := i.client.UpdateIssue(*updating); err != nil {
			return changed, err
		}
		issue = updating
	}

	// create/update children
	for _, child := range ticket.Children {
		childChanged, err := i.createTicket(projectID, issue.Id, child)
		if err != nil {
			return false, err
		}
		if childChanged {
			changed = true
		}
	}
	return changed, nil
}

func (i *Importer) mergeTickets(src *Ticket, dst *redmine.Issue) error {
	if src.Subject != nil {
		dst.Subject = *src.Subject
	}
	if src.Description != nil {
		dst.Description = *src.Description
	}
	if src.Tracker != nil {
		trackerID, err := i.trackers.findIDByName(*src.Tracker)
		if err != nil {
			return err
		}
		dst.TrackerId = trackerID
	}
	if src.Priority != nil {
		priorityID, err := i.priorities.findIDByName(*src.Priority)
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
