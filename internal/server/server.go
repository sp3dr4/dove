package server

import "context"

// Server represents a generic server that can be started and stopped
type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Addr() string
}
