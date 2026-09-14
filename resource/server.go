package resource

import (
	"errors"
)

type Server struct {
	name     string
	ip       string
	port     int
	cpuUsage float64
	running  bool
}

func (server *Server) Start() {
	server.running = true
}

func (server *Server) Stop() {
	server.running = false
}

func (server *Server) UpdateCPU(value float64) error {
	if value < 0 {
		return errors.New("Value cannot be set below 0.")
	}

	if value > 100 {
		return errors.New("Value cannot be set to over 100.")
	}

	server.cpuUsage = value

	return nil
}

func (server Server) IsHealthy() bool {
	return server.cpuUsage < 90 && server.running
}

func (server Server) ResourceName() string {
	return server.name
}

func NewServer(name string, ip string, port int) *Server {
	return &Server{
		name:     name,
		ip:       ip,
		port:     port,
		cpuUsage: 0,
		running:  false,
	}
}
