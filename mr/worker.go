package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// KeyValue is a slice returned by the Map function.
type KeyValue struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ByKey for sorting by key.
type ByKey []KeyValue

// Len for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// Worker is called by cmd/mrworker/main.go.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) error {
	for {
		task := CallAssignTask()

		switch task.Type {
		case MapTask:
			// NOTE: Map task logic:
			// 1. Reads the input file
			// 2. Calls mapf() to get KeyValue pairs
			// 3. Partitions keys into buckets using ihash(key) % nReduce
			// 4. Writes kvs files named mr-{mapTaskID}-{reduceID}
			// 5. Tells coordinator the task is complete

			// Read input file
			data, err := os.ReadFile(task.FileName)
			if err != nil {
				return fmt.Errorf("error reading input file: %w", err)
			}
			content := string(data)

			// Call mapf() on contents
			kvs := mapf(task.FileName, content)

			// Write kvs files
			buckets := make([][]KeyValue, task.NumReducers)

			for _, kv := range kvs {
				bucket := ihash(kv.Key) % task.NumReducers
				buckets[bucket] = append(buckets[bucket], kv)
			}

			for i, kvs := range buckets {
				oname := fmt.Sprintf("mr-%d-%d", task.ID, i)
				ofile, err := os.Create(oname)
				if err != nil {
					return fmt.Errorf("error creating intermediate file %s: %w", oname, err)
				}

				encoding := json.NewEncoder(ofile)
				for _, kv := range kvs {
					encoding.Encode((&kv))
				}

				ofile.Close()
			}

			// Tell coordinator we finished this map task
			CallFinishedTask(task)
			continue

		case ReduceTask:
			// NOTE: Reduce task logic:
			// 1. Finds corresponding intermediate files
			// 2. Decodes JSON-encoded intermeidate files as stream
			// 3. Partitions keys into buckets using ihash(key) % nReduce
			// 4. Writes kvs files named mr-{mapTaskID}-{reduceID}
			// 5. Tells coordinator the task is complete

			// NOTE: Worker can figure out what intermediate files to read based on task.mapTaskID
			// where "mr-*-{task.ID}"

			pattern := fmt.Sprintf("mr-*-%d", task.ID)
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return fmt.Errorf("error finding matches for pattern %q: %v", pattern, err)
			}

			// Read all kvs files and collect KeyValue pairs
			// NOTE: Each line in kvs file is a separate JSON-encoded KeyValue
			var kvs []KeyValue
			for _, file := range matches {
				f, err := os.Open(file)
				if err != nil {
					return fmt.Errorf("error opening intermediate file %s: %w", f.Name(), err)
				}

				decoder := json.NewDecoder(f)

				// Decode each JSON object one at a time
				for {
					var kv KeyValue
					if err := decoder.Decode(&kv); err != nil {
						if err == io.EOF {
							f.Close()
							break
						}
					}
					kvs = append(kvs, kv)
				}

				f.Close()
			}

			// Group all values by key
			sort.Sort(ByKey(kvs))

			// For each unique key, call reducef(key, []values)
			oname := fmt.Sprintf("mr-out-%d", task.ID)
			ofile, _ := os.Create(oname)
			defer ofile.Close()

			i := 0
			for i < len(kvs) {
				j := i + 1
				for j < len(kvs) && kvs[j].Key == kvs[i].Key {
					j++
				}
				values := []string{}
				for k := i; k < j; k++ {
					values = append(values, kvs[k].Value)
				}
				output := reducef(kvs[i].Key, values)

				// this is the correct format for each line of Reduce output.
				fmt.Fprintf(ofile, "%v %v\n", kvs[i].Key, output)

				i = j
			}

			CallFinishedTask(task)
			continue

		case WaitTask:
			// No tasks ready yet, sleep
			time.Sleep(10 * time.Second)
			continue

		case DoneTask:
			// All work complete, exit
			os.Exit(0)
		}
	}
}

func CallAssignTask() *AssignTaskReply {
	args := AssignTaskArgs{}
	reply := AssignTaskReply{}

	ok := call("Coordinator.AssignTask", &args, &reply)
	if !ok {
		fmt.Printf("call failed!\n")
	}

	return &reply
}

func CallFinishedTask(task *AssignTaskReply) *FinishedTaskReply {
	args := FinishedTaskArgs{}
	reply := FinishedTaskReply{}
	args.ID = task.ID
	args.Type = task.Type

	ok := call("Coordinator.FinishedTask", &args, &reply)
	if !ok {
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
