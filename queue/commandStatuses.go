package queue

/*
CommandStatusQueued is the status abstraction of a command
*/
const (
	Queued = iota
	Executing
	Success
	Error
	OutputMismatch
)
