package queue

//Command is a struct abstracting a command in a command queue
type Command struct {
	Command 		string
	ExpectedOutput 	string
	Output  		string
	Status  		int
}
