// Package mr is the library code for the distributed MapReduce
package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
)

type Coordinator struct {
	MapTasks    []Task
	ReduceTasks []Task
	mu          sync.Mutex
}

type Task struct {
	ID       int
	Type     TaskType
	State    TaskState
	FileName string
}

// RPC handlers for the workers to call

func (c *Coordinator) allMapTasksCompleted() bool {
	allMapTasksCompleted := true
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State != Completed {
			allMapTasksCompleted = false
			break
		}
	}

	return allMapTasksCompleted
}

func (c *Coordinator) allTasksIdle() bool {
	allTasksIdle := true

	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State != Idle || task.State != Completed {
			allTasksIdle = false
			return allTasksIdle
		}
	}

	for i := range c.ReduceTasks {
		task := &c.ReduceTasks[i]

		if task.State != Idle || task.State != Completed {
			allTasksIdle = false
			return allTasksIdle
		}
	}

	return allTasksIdle
}

func (c *Coordinator) allTasksCompleted() bool {
	done := true
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State != Completed {
			done = false
			return done
		}
	}

	for i := range c.ReduceTasks {
		task := &c.ReduceTasks[i]

		if task.State != Completed {
			done = false
			return done
		}
	}

	return done
}

func (c *Coordinator) AssignTask(args *AssignTaskArgs, reply *AssignTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Assign a MapTask when there's an idle MapTask
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State == Idle {
			task.State = InProgress
			// TODO: add deadline for worker timeout detection

			reply.Type = MapTask
			reply.ID = task.ID
			reply.FileName = task.FileName
			reply.NumReducers = len(c.ReduceTasks)

			return nil
		}
	}
	// Assign a ReduceTask when all map tasks are completed AND there's an idle reduce task
	if c.allMapTasksCompleted() {
		for i := range c.ReduceTasks {
			task := &c.ReduceTasks[i]

			if task.State == Idle {
				task.State = InProgress
				reply.Type = ReduceTask
				reply.ID = task.ID

				return nil
			}
		}
	}

	// Assign a WaitTask if no idle tasks exist but work is still in progress
	if c.allTasksIdle() {
		reply.Type = WaitTask
		return nil
	}

	// Assign a DoneTask when all map AND all reduce tasks are completed
	if c.allTasksCompleted() {
		reply.Type = DoneTask
		return nil
	}

	// NOTE: if we get here maybe something went wrong, may want to return a Error reply.Type
	return nil
}

func (c *Coordinator) FinishedTask(args *FinishedTaskArgs, reply *FinishedTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// TODO: #1 This function needs proper task completion tracking
	if args.Type == MapTask {
		for i := range len(c.MapTasks) {
			task := &c.MapTasks[i]

			if task.ID == args.ID {
				task.State = Completed
				// TODO: #5 Clear any deadline/timeout for this task
				break
			}
		}
	} else if args.Type == ReduceTask {
		for i := range len(c.ReduceTasks) {
			task := &c.ReduceTasks[i]

			if task.ID == args.ID {
				task.State = Completed
			}
		}
	}

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

// Done is called by cmd/mrcoordinator/main.go periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
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
// cmd/mrcoordinator/main.go calls this function.
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
