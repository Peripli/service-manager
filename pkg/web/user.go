package web

// UserContext holds the information for the current user
type UserContext struct {
	Data

	Name string
}

//go:generate counterfeiter . Data
type Data interface {
	// Data reads the additional data from the context into the specified struct
	Data(v interface{}) error
}
