package mr

import "testing"

func TestInitialMapTaskAssignment(t *testing.T) {
	files := []string{"fileA.txt", "fileB.txt"}
	coordinator := MakeCoordinator(files, 1)

	// simulate worker's call
	args := AssignTaskArgs{}
	reply := AssignTaskReply{}
	coordinator.AssignTask(&args, &reply)

	// verify
	if reply.TaskType != MapTask {
		t.Fatalf("Expected a Map task, but got %v", reply.TaskType)
	}

	if reply.FileName != "fileA.txt" {
		t.Fatalf("Expected fileA.txt but got %v", reply.FileName)
	}

	if reply.TaskID != 0 {
		t.Fatalf("Expected TaskID 0, but got %d", reply.TaskID)
	}
}

func TestCallAssignTaskViaRPC(t *testing.T) {
	files := []string{"fileA.txt", "fileB.txt"}
	MakeCoordinator(files, 1)

	// MakeCoordinator already starts the RPC server
	args := AssignTaskArgs{}
	reply := AssignTaskReply{}

	ok := call("Coordinator.AssignTask", &args, &reply)

	if !ok {
		t.Fatal("RPC call failed")
	}

	// verify
	if reply.TaskType != MapTask {
		t.Fatalf("Expected a Map task, but got %v", reply.TaskType)
	}

	if reply.FileName != "fileA.txt" {
		t.Fatalf("Expected fileA.txt but got %v", reply.FileName)
	}

	if reply.TaskID != 0 {
		t.Fatalf("Expected TaskID 0, but got %d", reply.TaskID)
	}
}
