package sync

import (
	"io"
	"os"

	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/uphy/go-redmine"
)

type Sync struct {
	client    *redmine.Client
	Converter *Converter
	logger    *log.Logger
}

func New(endpoint string, apiKey string) (*Sync, error) {
	client := redmine.NewClient(endpoint, apiKey)
	logger := log.New(os.Stderr, "[sync]", log.LstdFlags|log.Lmicroseconds)
	return &Sync{
		client:    client,
		Converter: newConverter(client),
		logger:    logger,
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

func (s *Sync) Watch(file string, ignoreImportError bool) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	if err := w.Add(file); err != nil {
		return err
	}

	fileForRead, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fileForRead.Close()

	config, err := s.Converter.ReadConfig(fileForRead)
	if err != nil {
		return err
	}
	for evt := range w.Events {
		if evt.Op != fsnotify.Write {
			continue
		}
		s.logger.Println("Detected file modification.")
		if _, err := fileForRead.Seek(0, 0); err != nil {
			return err
		}
		config2, err := s.Converter.ReadConfig(fileForRead)
		if err != nil {
			return err
		}
		s.logger.Println("Importing the changes...")
		changed, err := s.Import(config2, config)
		if err != nil {
			if ignoreImportError {
				s.logger.Printf("Failed to import: %s", err)
				continue
			}
			return err
		}
		if changed {
			s.logger.Println("Rewriting the config file...")
			f, err := os.Create(file)
			if err != nil {
				return err
			}
			if err := s.Converter.SaveConfig(f, config2); err != nil {
				return err
			}
			f.Close()
		}
		config = config2
		s.logger.Println("Successfully applied the changes.")
	}
	return nil
}

func (s *Sync) Import(config *Config, base *Config) (changed bool, err error) {
	changes, err := DiffTickets(s.Converter, base, config)
	if err != nil {
		return false, err
	}

	changed = false
	for _, change := range changes {
		switch change.Change {
		case ChangeAdded, ChangeUpdated:
			s.logger.Printf("Updating issue #%d...", change.Ticket2.ID)
			ticketChanged, err := s.createTicket(change.Ticket2)
			if err != nil {
				return false, err
			}
			if ticketChanged {
				changed = true
			}
		}
	}
	return
}

func (s *Sync) ImportFile(file *os.File, base *os.File) (config *Config, changed bool, err error) {
	var configBase *Config
	if base != nil {
		c, err := s.Converter.ReadConfig(base)
		if err != nil {
			return nil, false, err
		}
		configBase = c
	} else {
		configBase = &Config{}
	}
	config, err = s.Converter.ReadConfig(file)
	if err != nil {
		return nil, false, err
	}

	changed, err = s.Import(config, configBase)
	if err != nil {
		return nil, false, err
	}
	return config, changed, err
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
	return changed, nil
}
