package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

type AssignTaskArgs struct {
	// TODO: add fields (e.g. WorkerID)
}

type TaskType int

const (
	MapTask TaskType = iota
	ReduceTask
	WaitTask
	DoneTask
)

type AssignTaskReply struct {
	TaskType    TaskType
	TaskID      int
	FileName    string // for map tasks
	NumReducers int    // for map tasks
}

type TaskState int

const (
	Idle TaskState = iota
	InProgress
	Completed
)

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
