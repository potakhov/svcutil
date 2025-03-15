package svcutil

type ServiceID struct {
	Hostname string
	Service  string
	ID       int
}

func (sid ServiceID) String() string {
	return sid.Service
}

func (sid ServiceID) Int() int {
	return sid.ID
}
