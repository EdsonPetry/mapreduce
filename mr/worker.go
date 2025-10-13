package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"time"
)

// KeyValue is a slice returned by the Map function.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Worker is called by cmd/mrworker/main.go.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	task := CallAssignTask()

	switch task.Type {
	case MapTask:
		// Read input file
		data, err := os.ReadFile(task.FileName)
		check(err)
		content := string(data)

		// Call mapf() on contents
		intermediate := mapf(task.FileName, content)
		fmt.Printf("intermediate: %v\n", intermediate)

		// Write intermediate files
		buckets := make([][]KeyValue, task.NumReducers)

		for _, kv := range intermediate {
			bucket := ihash(kv.Key) % task.NumReducers
			buckets[bucket] = append(buckets[bucket], kv)
		}

		for i, kvs := range buckets {
			oname := fmt.Sprintf("mr-%d-%d", task.ID, i)
			ofile, err := os.Create(oname)
			check(err)

			encoding := json.NewEncoder(ofile)
			for _, kv := range kvs {
				encoding.Encode((&kv))
			}

			ofile.Close()
		}

		// Tell coordinator we finished this map task
		CallFinishedTask()

	case ReduceTask:
	// Read all intermediate files
	// Call reducef() on data
	// Write final output file
	// Tell coordinator we finished this reduce task
	case WaitTask:
		// No tasks ready yet, sleep
		time.Sleep(10 * time.Second)
	case DoneTask:
		// All work complete, exit
	}
}

func CallAssignTask() *AssignTaskReply {
	args := AssignTaskArgs{}
	reply := AssignTaskReply{}

	ok := call("Coordinator.AssignTask", &args, &reply)
	if ok {
		// reply will contain TaskType, TaskID, Filename
		fmt.Printf("Got task: %v\n", reply)
	} else {
		fmt.Printf("call failed!\n")
	}

	return &reply
}

func CallFinishedTask() *FinishedTaskReply {
	args := FinishedTaskArgs{}
	reply := FinishedTaskReply{}

	ok := call("Coordinator.FinishedTask", &args, &reply)
	if ok {
		fmt.Printf("Finished task: %v\n", reply)
	} else {
		fmt.Println("call failed!")
	}

	return &reply
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args, reply any) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
