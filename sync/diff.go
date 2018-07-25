package sync

type (
	Change      string
	IssueChange struct {
		Ticket1 *Ticket
		Ticket2 *Ticket
		Change  Change
	}
)

const (
	ChangeRemoved Change = "removed"
	ChangeAdded   Change = "added"
	ChangeUpdated Change = "updated"
)

func DiffTickets(converter *Converter, config1 *Config, config2 *Config) ([]IssueChange, error) {
	ticketMap := func(config *Config) (map[int]*Ticket, error) {
		if config == nil || config.Projects == nil {
			return map[int]*Ticket{}, nil
		}
		tickets, err := converter.toFlat(config)
		if err != nil {
			return nil, err
		}

		m := map[int]*Ticket{}
		for _, t := range tickets {
			m[t.ID] = t
		}
		return m, nil
	}
	tickets1, err := ticketMap(config1)
	if err != nil {
		return nil, err
	}
	tickets2, err := ticketMap(config2)
	if err != nil {
		return nil, err
	}

	changes := []IssueChange{}
	for _, t1 := range tickets1 {
		t2, ok := tickets2[t1.ID]
		if ok {
			if !equals(t1, t2) {
				changes = append(changes, IssueChange{t1, t2, ChangeUpdated})
			}
		} else {
			changes = append(changes, IssueChange{t1, t2, ChangeRemoved})
		}
	}
	for _, t2 := range tickets2 {
		_, ok := tickets1[t2.ID]
		if !ok {
			changes = append(changes, IssueChange{nil, t2, ChangeAdded})
		}
	}
	return changes, nil
}

func equals(t1 *Ticket, t2 *Ticket) bool {
	if !equalsString(t1.Subject, t2.Subject) {
		return false
	}
	if !equalsString(t1.Description, t2.Description) {
		return false
	}
	if t1.ParentID != t2.ParentID {
		return false
	}
	if !equalsString(t1.Priority, t2.Priority) {
		return false
	}
	if !equalsString(t1.Project, t2.Project) {
		return false
	}
	if !equalsString(t1.DueDate, t2.DueDate) {
		return false
	}
	if !equalsString(t1.StartDate, t2.StartDate) {
		return false
	}
	if !equalsString(t1.Status, t2.Status) {
		return false
	}
	if !equalsString(t1.Tracker, t2.Tracker) {
		return false
	}
	if !equalsString(t1.Assignee, t2.Assignee) {
		return false
	}
	return true
}

func equalsString(s1, s2 *string) bool {
	if s1 == nil && s2 == nil {
		return true
	}
	if s1 != nil && s2 != nil {
		return *s1 == *s2
	}
	return false
}
