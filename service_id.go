package svcutil

import "fmt"

type ServiceID struct {
	Hostname string
	Service  string
	ID       int
}

func NewServiceID(service string, id int) ServiceID {
	var sid ServiceID
	sid.Hostname = Hostname()
	sid.ID = id

	if id > 0 {
		sid.Service = fmt.Sprintf("%s-%s-%d", sid.Hostname, service, id)
	} else {
		sid.Service = fmt.Sprintf("%s-%s", sid.Hostname, service)
	}

	return sid
}

func (sid ServiceID) String() string {
	return sid.Service
}

func (sid ServiceID) Int() int {
	return sid.ID
}
