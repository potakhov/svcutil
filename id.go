package svcutil

type ID struct {
	Hostname string
	Service  string
	ID       int
}

func (sid ID) String() string {
	return sid.Service
}

func (sid ID) Int() int {
	return sid.ID
}
