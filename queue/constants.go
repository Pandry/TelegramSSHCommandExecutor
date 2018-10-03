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

/*
Those constants are used to indicate how a command failure should be handled
*/
const (
	Ignore = iota
	Retry	
	Interrupt
)
