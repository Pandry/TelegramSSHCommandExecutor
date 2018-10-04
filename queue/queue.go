package queue

import (
	"errors"

	"../utils"

	"../config"
)

//Queue is a struct containing the first command pointer and the count
//Emulates a list
type Queue struct {
	commandQueue []Command
	running      int //Index
	onFail       int
}

const _CommandStatusSeparator = "` `|` `"

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
		q.onFail = Ignore
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
			cres = " 🕐  Queued         "
			break
		case Executing:
			cres = " ⚙️  Executing      "
			break
		case Success:
			cres = "✅  Success          "
			break
		case Error:
			cres = " ❌  Error           "
			break
		case OutputMismatch:
			cres = " ❗️  Output Mismatch"
			break
		}
		cres = _CommandStatusSeparator + cres
		avSpace := config.Conf.Settings.MaxMessageColumns - len(utils.RemoveMarkdownSyntax(cres)) //Emojis count twice
		var commandText string
		commandText = "`" + cmd.Command
		if len(cmd.Command)-3 > avSpace {
			commandText = cmd.Command[0 : len(cmd.Command)-avSpace-1-3]
			commandText = "`" + commandText + "...`"
		} else {
			for i := 0; i < avSpace-len(cmd.Command)+len(utils.RemoveMarkdownSyntax(cres)); i++ {
				commandText += " "
			}
			commandText += "`"
		}
		cres = commandText + cres
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

//GetActualCommand returns the command that should be in execution without putting it into a executing state
func (q *Queue) GetActualCommand() (string, error) {
	if q.running+1 > len(q.commandQueue) {
		return "", errors.New("Commands are finished")
	}
	//If is queued, returns next one
	if q.commandQueue[q.running].Status != Queued {
		return q.commandQueue[q.running].Command, errors.New("Command were already Executed")
	}
	return q.commandQueue[q.running].Command, nil
}

//GetActualCommandAndExecute returns the command that should be in execution without putting it into a executing state
func (q *Queue) GetActualCommandAndExecute(force bool) (string, error) {
	if q.running+1 > len(q.commandQueue) {
		return "", errors.New("Commands are finished")
	}
	//If is queued, returns next one
	if !force {
		if q.commandQueue[q.running].Status != Queued {
			return q.commandQueue[q.running].Command, errors.New("Command were already Executed")
		}
	} else {
		q.commandQueue[q.running].Status = Executing
	}
	return q.commandQueue[q.running].Command, nil
}

//PopCommand returns the next command to execute and put it in a executing state
func (q *Queue) PopCommand() (string, error) {
	q.running++
	if q.running+1 > len(q.commandQueue) {
		return "", errors.New("Commands are finished")
	}
	//If is queued, returns next one
	if q.commandQueue[q.running].Status != Queued {
		return q.commandQueue[q.running].Command, errors.New("Command were already Executed")
	}
	q.commandQueue[q.running].Status = Executing
	return q.commandQueue[q.running].Command, nil
}

//IsOver returns a bool that indicates if the queue is over or there are other commans to execute
func (q *Queue) IsOver() bool {
	if q.running+2 > len(q.commandQueue) {
		return true
	}
	return false
}

//IncrementQueue returns a bool that indicates if the queue is over or there are other commans to execute
func (q *Queue) IncrementQueue() bool {
	if q.running+2 > len(q.commandQueue) {
		return false
	}
	q.running++
	return true
}

//GetCommandStatus returns the actual command status
//WARNING - returns the Success status if the queque is not started yet
func (q *Queue) GetCommandStatus() int {
	if q.running == -1 {
		return Success
	}
	return q.commandQueue[q.running].Status
}

//IsRetryAllowed returns a bool that indicates if it's allowed to retry the command
func (q *Queue) IsRetryAllowed() bool {
	return q.onFail == Retry
}

//ShuldIgnoreError returns a bool that indicates if the execution should ignore if an eventual error occours
func (q *Queue) ShuldIgnoreError() bool {
	return q.onFail == Ignore
}

//ShuldRetryOnError returns a bool that indicates if the execution should retry the command when an error occours
func (q *Queue) ShuldRetryOnError() bool {
	return q.onFail == Retry
}

//ShuldQuitOnError returns a bool that indicates if the execution should interrupt when an error occours
func (q *Queue) ShuldQuitOnError() bool {
	return q.onFail == Interrupt
}

//SetOnFail takes a int (consult the constants file) in input that tells if it's allowed to retry the command
func (q *Queue) SetOnFail(f int) {
	q.onFail = f
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
