package sync

import (
	"io"
	"os"

	"github.com/mattn/go-redmine"
)

type Sync struct {
	client    *redmine.Client
	Converter *Converter
}

func New(endpoint string, apiKey string) (*Sync, error) {
	client := redmine.NewClient(endpoint, apiKey)
	return &Sync{
		client:    client,
		Converter: newConverter(client),
	}, nil
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

	return s.Converter.Convert(issues)
}

func (s *Sync) Import(file *os.File, base *os.File) (config *Config, changed bool, err error) {
	var configBase *Config
	if base != nil {
		c, err := s.Converter.ReadConfig(base)
		if err != nil {
			return nil, false, err
		}
		configBase = c
	} else {
		configBase = &Config{}
		configBase.Projects = []*Project{}
	}
	config, err = s.Converter.ReadConfig(file)
	if err != nil {
		return nil, false, err
	}

	changes, err := DiffTickets(s.Converter, configBase, config)
	if err != nil {
		return nil, false, err
	}

	changed = false
	for _, change := range changes {
		switch change.change {
		case ChangeAdded, ChangeUpdated:
			ticketChanged, err := s.createTicket(change.ticket2)
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

func (s *Sync) createTicket(ticket *Ticket) (bool, error) {
	var issue *redmine.Issue
	changed := false
	if ticket.ID == 0 {
		// create
		issue = &redmine.Issue{}
		if err := s.Converter.mergeTicketToIssue(ticket, issue); err != nil {
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
		if err := s.Converter.mergeTicketToIssue(ticket, updating); err != nil {
			return changed, err
		}
		if err := s.client.UpdateIssue(*updating); err != nil {
			return changed, err
		}
		issue = updating
	}

	// create/update children
	for _, child := range ticket.Children {
		childChanged, err := s.createTicket(child)
		if err != nil {
			return false, err
		}
		if childChanged {
			changed = true
		}
	}
	return changed, nil
}
