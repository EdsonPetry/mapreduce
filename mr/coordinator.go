// Package mr is the library code
package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

type Coordinator struct {
	MapTasks    []Task
	ReduceTasks []Task
}

type Task struct {
	ID       int
	Type     TaskType
	State    TaskState
	FileName string
}

// RPC handlers for the workers to call

func (c *Coordinator) AssignTask(args *AssignTaskArgs, reply *AssignTaskReply) error {
	// find idle Map task.
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State == Idle {
			task.State = InProgress
			// TODO: add deadline

			reply.Type = MapTask
			reply.ID = task.ID
			reply.FileName = task.FileName
			reply.NumReducers = len(c.ReduceTasks)

			return nil
		}
	}

	for i := range c.ReduceTasks {
		task := &c.ReduceTasks[i]

		if task.State == Idle {
			task.State = InProgress

			reply.Type = ReduceTask
			reply.ID = task.ID
			reply.FileName = task.FileName
		}
	}

	// no idle map tasks found, for now we'll just return
	return nil
}

func (c *Coordinator) FinishedTask(args *FinishedTaskArgs, reply *FinishedTaskReply) error {
	// find finished task
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.ID == args.ID {
			task.State = Completed
		}
	}

	// Mark non-assigned ReduceTask with filename of intermediate file
	for i := range c.ReduceTasks {
		task := &c.ReduceTasks[i]

		if task.FileName != "" && task.State == Idle {
			task.FileName = args.FileName // for locating intermediate file
			break
		}
	}

	return nil
}

func (c *Coordinator) WaitTask(args *WaitTaskArgs, reply *WaitTaskReply) *WaitTaskArgs {
	return nil
}

func (c *Coordinator) DoneTask(args *DoneTaskArgs, reply *DoneTaskReply) *DoneTaskReply {
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	// l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	go http.Serve(l, nil)
}

// Done is called by main/mrcoordinator.go periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := true

	for i := range c.ReduceTasks {
		task := &c.ReduceTasks[i]

		if task.State != Completed {
			ret = false
		}
	}

	return ret
}

// MakeCoordinator creates a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	for i, file := range files {
		task := Task{i, MapTask, Idle, file}
		c.MapTasks = append(c.MapTasks, task)
	}

	for i := range nReduce {
		task := Task{i, ReduceTask, Idle, ""}
		c.ReduceTasks = append(c.ReduceTasks, task)
	}

	c.server()
	return &c
}
