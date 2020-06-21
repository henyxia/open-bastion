package service

// Service represents all services ran by the bastion
type Service interface {
	InitOnly() bool
	Init() error
	Start() error
	Stop() error
}
