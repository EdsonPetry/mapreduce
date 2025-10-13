package mr

import "testing"

func TestMapTaskAssignment(t *testing.T) {
	files := []string{"fileA.txt", "fileB.txt"}
	coordinator := MakeCoordinator(files, 1)

	// simulate worker's call
	args := AssignTaskArgs{}
	reply := AssignTaskReply{}
	coordinator.AssignTask(&args, &reply)

	if reply.Type != MapTask {
		t.Fatalf("Expected a Map task, but got %v", reply.Type)
	}

	if reply.FileName != "fileA.txt" {
		t.Fatalf("Expected fileA.txt but got %v", reply.FileName)
	}

	if reply.ID != 0 {
		t.Fatalf("Expected TaskID 0, but got %d", reply.ID)
	}

	if reply.NumReducers != 1 {
		t.Fatalf("Expected NumReducers 1, but got %d", reply.NumReducers)
	}
}

func TestMapTaskIntermediateFiles(t *testing.T) {
	// 1. Create a temporary input file with simple test data
	//    - Use os.CreateTemp() or write to a known temp location
	//    - Content suggestion: "hello world hello" (tests word frequency)
	//    - Remember the filename to pass to coordinator

	// 2. Set up coordinator with the test file
	//    - MakeCoordinator([]string{tempFileName}, nReduce)
	//    - Choose nReduce = 2 or 3 to test partitioning across multiple files

	// 3. Create a simple word-count map function for testing
	//    - Copy logic from mrapps/wc.go Map function (lines 19-31)
	//    - Or import and reuse if possible
	//    - Should split text into words and emit KeyValue{word, "1"}

	// 4. Simulate the worker's map task execution
	//    - Call coordinator.AssignTask() to get a task assignment
	//    - Read the input file
	//    - Call the map function on file contents
	//    - Partition results into buckets using ihash(key) % nReduce
	//    - Write intermediate files: mr-{taskID}-{reduceID}
	//    - Each file should contain JSON-encoded KeyValue pairs (one per line)

	// 5. Verify intermediate file output
	//    - Check that the expected number of mr-*-* files exist
	//    - For each file:
	//      a) Open and parse JSON (use json.Decoder, read line by line)
	//      b) Verify each line is a valid KeyValue struct
	//      c) Verify keys in this file hash to the correct bucket
	//    - Verify all expected words appear across all files
	//    - For "hello world hello": expect "hello" with count 2, "world" with count 1

	// 6. Clean up temporary files
	//    - Remove temp input file
	//    - Remove all mr-*-* intermediate files
	//    - Use t.Cleanup() or defer for reliable cleanup
}
