package backend

// Conn represents a connection to a target
type Conn struct {
	Command string
	User    string
	Host    string
	Port    int
}
