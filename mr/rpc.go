package mr

//
// RPC definitions.
//

import (
	"os"
	"strconv"
)

type TaskState int

const (
	Idle TaskState = iota
	InProgress
	Completed
)

type TaskType int

const (
	MapTask TaskType = iota
	ReduceTask
	WaitTask
	DoneTask
)

type AssignTaskArgs struct{}

type AssignTaskReply struct {
	Type        TaskType
	State       TaskState
	ID          int
	FileName    string // for map tasks
	NumReducers int    // for map tasks
}

type FinishedTaskArgs struct {
	ID   int
	Type TaskType
}

type FinishedTaskReply struct {
	Type  TaskType
	State TaskState
	ID    int
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
