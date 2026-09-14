package resource

import "errors"

type Worker struct {
	id      string
	name    string
	section string
	onLeave bool
}

func (worker Worker) IsOnLeave() bool {
	return worker.onLeave
}

func (worker Worker) ResourceName() string {
	return worker.name
}

func (worker *Worker) OnLeave() {
	worker.onLeave = true
}

func (worker *Worker) OffLeave() {
	worker.onLeave = false
}

func (worker *Worker) UpdateSection(position string) error {
	if worker.section == " " {
		return errors.New("Section cannot be empty")
	}

	worker.section = position
	return nil
}

func NewWorker(id string, name string, section string, onLeave bool) *Worker {
	return &Worker{
		id:      id,
		name:    name,
		section: "",
		onLeave: false,
	}
}
