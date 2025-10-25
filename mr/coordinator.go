// Package mr is the library code for the distributed MapReduce
package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type Coordinator struct {
	MapTasks    []Task
	ReduceTasks []Task
	mu          sync.Mutex
}

type Task struct {
	ID        int
	Type      TaskType
	State     TaskState
	FileName  string
	Timestamp time.Time
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

func (c *Coordinator) allTasksCompleted() bool {
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State != Completed {
			return false
		}
	}

	for i := range c.ReduceTasks {
		task := &c.ReduceTasks[i]

		if task.State != Completed {
			return false
		}
	}

	return true
}

func (c *Coordinator) AssignTask(args *AssignTaskArgs, reply *AssignTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For each InProgress task, check if (current time - assigned time) > timeout (10s)
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State == InProgress && time.Since(task.Timestamp) > (10*time.Second) {
			// reset task, assuming worker is dead
			task.State = Idle
		}
	}

	for i := range c.ReduceTasks {
		task := &c.ReduceTasks[i]

		if task.State == InProgress && time.Since(task.Timestamp) > (10*time.Second) {
			task.State = Idle
		}
	}

	// Assign a MapTask when there's an idle MapTask
	for i := range c.MapTasks {
		task := &c.MapTasks[i]

		if task.State == Idle {
			task.State = InProgress
			task.Timestamp = time.Now()

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
				task.Timestamp = time.Now()

				reply.Type = ReduceTask
				reply.ID = task.ID

				return nil
			}
		}
	}

	// Assign a DoneTask when all map AND all reduce tasks are completed
	if c.allTasksCompleted() {
		reply.Type = DoneTask
		return nil
	}

	// No tasks are ready, tell worker to wait
	reply.Type = WaitTask
	return nil
}

func (c *Coordinator) FinishedTask(args *FinishedTaskArgs, reply *FinishedTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if args.Type == MapTask {
		for i := range len(c.MapTasks) {
			task := &c.MapTasks[i]

			if task.ID == args.ID {
				task.State = Completed
				task.Timestamp = time.Time{}
				break
			}
		}
	} else if args.Type == ReduceTask {
		for i := range len(c.ReduceTasks) {
			task := &c.ReduceTasks[i]

			if task.ID == args.ID {
				task.State = Completed
				task.Timestamp = time.Time{}
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
		task := Task{i, MapTask, Idle, file, time.Time{}}
		c.MapTasks = append(c.MapTasks, task)
	}

	for i := range nReduce {
		task := Task{i, ReduceTask, Idle, "", time.Time{}}
		c.ReduceTasks = append(c.ReduceTasks, task)
	}

	c.server()
	return &c
}
