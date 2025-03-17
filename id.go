package svcutil

import "fmt"

type ID struct {
	Hostname string
	Service  string

	Value string
	ID    int
}

func NewID(id int, service string) ID {
	sid := ID{
		Hostname: Hostname(),
		ID:       id,
		Service:  service,
	}

	if sid.ID > 0 {
		sid.Value = fmt.Sprintf("%s-%s-%d", sid.Hostname, service, sid.ID)
	} else {
		sid.Value = fmt.Sprintf("%s-%s", sid.Hostname, service)
	}

	return sid
}

func (sid ID) String() string {
	return sid.Value
}

func (sid ID) Int() int {
	return sid.ID
}

func (sid ID) Mask(mask string) string {
	if sid.ID > 0 {
		return fmt.Sprintf("%s-%s-%d", mask, sid.Service, sid.ID)
	} else {
		return fmt.Sprintf("%s-%s", mask, sid.Service)
	}
}
