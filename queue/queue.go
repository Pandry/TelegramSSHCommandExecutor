package queue

import "errors"

//Queue is a struct containing the first command pointer and the count
//Emulates a list
type Queue struct {
	commandQueue []Command
	running      int //Index
}

//AddCommand is used for adding a command to the linked list
func (q *Queue) AddCommand(cmd string) {
	if q.commandQueue == nil {

		q.running = -1 //None running
	}
	q.commandQueue = append(q.commandQueue, Command{
		Command: cmd,
		Status:  Queued,
	})
}

//AddCommandAndExpOut is used for adding a command and its correspondig expected output to the linked list
func (q *Queue) AddCommandAndExpOut(cmd string, exout string) {
	if q.commandQueue == nil {
		q.running = -1 //None running
	}
	q.commandQueue = append(q.commandQueue, Command{
		Command:        cmd,
		ExpectedOutput: exout,
		Status:         Queued,
	})

}

//GetScriptsStatus a SMS-ready array for jobs statuses queue
func (q *Queue) GetScriptsStatus() []string {
	var res []string
	for _, cmd := range q.commandQueue {
		var cres string
		switch cmd.Status {
		//✅🕐⚙️❌❗️
		case Queued:
			cres = "🕐  Queued          - "
			break
		case Executing:
			cres = "⚙️  Executing       - "
			break
		case Success:
			cres = "✅  Success         - "
			break
		case Error:
			cres = "❌  Error           - "
			break
		case OutputMismatch:
			cres = "❗️  Output Mismatch - "
			break
		}
		cres += cmd.Command
		res = append(res, cres)
	}
	return res
}

//GetExpectedOutput returns the current command queue
func (q *Queue) GetExpectedOutput() string {
	return q.commandQueue[q.running].ExpectedOutput
}

//GetCommandQueue returns the expected output for the current command
func (q *Queue) GetCommandQueue() []Command {
	return q.commandQueue
}

//AddBulkCommands is used for adding multiple commands from a string array to the linked list
func (q *Queue) AddBulkCommands(cmds []string) {
	for _, cmd := range cmds {
		q.AddCommand(cmd)
	}
}

//AddBulkCommandsAndOutput is used for adding multiple commands and their expected output from a string array to the linked list
func (q *Queue) AddBulkCommandsAndOutput(cmds []string, outs []string) {

	for i, cmd := range cmds {
		if len(outs) >= i+1 && outs[i] != "" {
			q.AddCommandAndExpOut(cmd, outs[i])
		} else {
			q.AddCommandAndExpOut(cmd, "")
		}
	}
}

//GetQueueLength returns the length of the queue
func (q *Queue) GetQueueLength() int {
	return len(q.commandQueue)
}

//GetNextCommandToExecute returns the next command to execute and put it in a executing state
func (q *Queue) GetNextCommandToExecute() (string, error) {
	q.running++
	if q.running+1 > len(q.commandQueue) {
		return "", errors.New("Commands are finished")
	}
	//If is queued, returns next one
	if q.commandQueue[q.running].Status != Queued {
		return "", errors.New("Command were already Executed")
	}
	q.commandQueue[q.running].Status = Executing
	return q.commandQueue[q.running].Command, nil
}

/*
//GetCommandToExecute returns the command to execute and put it in a executing state
//it also edits the running command the the returned one
func (q *Queue) GetCommandToExecute(index int) (string, error) {
	//If the index is out of range
	if len(q.commandQueue)-1 > index {
		//Runs the latest one
		if q.commandQueue[len(q.commandQueue)-1].status == Queued {
			q.commandQueue[len(q.commandQueue)-1].status = Executing
			return q.commandQueue[len(q.commandQueue)-1].command, errors.New("Index out of range exception")
		}
		return "", errors.New("Index out of range exception")
	}
	if q.commandQueue[index].status == Queued {
		q.commandQueue[index].status = Executing
		return q.commandQueue[index].command, nil
	} else {
		//not necessarly last one is to execute
		return "", errors.New("Command already execuded")
	}
}
*/

//SetCommandError sets the command status to error and writes its error status to the output command
func (q *Queue) SetCommandError(err error) {
	q.commandQueue[q.running].Output = err.Error()
	q.commandQueue[q.running].Status = Error
}

//SetCommandOutput sets the executing command's output
func (q *Queue) SetCommandOutput(output string) {
	q.commandQueue[q.running].Output = output
	q.commandQueue[q.running].Status = Success
}

//SetCommandOutputMismatch sets the executing command's output
func (q *Queue) SetCommandOutputMismatch(output string) {
	q.commandQueue[q.running].Output = output
	q.commandQueue[q.running].Status = OutputMismatch
}
