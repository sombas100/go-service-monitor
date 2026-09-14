package resource

import (
	"errors"
)

type Database struct {
	name        string
	connections int
	running     bool
}

func (database Database) IsHealthy() bool {
	return database.connections < 100 && database.running
}

func (database Database) ResourceName() string {
	return database.name
}

func (database *Database) Start() {
	database.running = true
}

func (database *Database) Stop() {
	database.running = false
}

func (database *Database) UpdateConnection(value int) error {
	if database.connections < 0 {
		return errors.New("Connections cannot be set below 0")
	}

	database.connections = value
	return nil
}

func NewDatabase(name string, connections int) *Database {
	return &Database{
		name:        name,
		connections: 0,
		running:     false,
	}
}
