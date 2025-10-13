package mr

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode"
)

// simpleMap function defined at package level for reuse
func simpleMap(filename string, contents string) []KeyValue {
	// function to detect word separators.
	ff := func(r rune) bool { return !unicode.IsLetter(r) }

	// split contents into an array of words.
	words := strings.FieldsFunc(contents, ff)

	kva := []KeyValue{}
	for _, w := range words {
		kv := KeyValue{w, "1"}
		kva = append(kva, kv)
	}
	return kva
}

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
	// Create temporary input file with content "hello world hello"
	tempf, err := os.CreateTemp("", "fileA.txt")
	if err != nil {
		t.Fatalf("Error when creating temp file: %v", err)
	}
	defer os.Remove(tempf.Name())

	if _, err := tempf.Write([]byte("hello world hello")); err != nil {
		t.Fatalf("Error when writing to temp file: %v", err)
	}
	tempf.Close()

	MakeCoordinator([]string{tempf.Name()}, 1)

	// Create channel for synchronization between test and worker goroutine
	done := make(chan bool)

	// Launch Worker in goroutine with channel communication
	go func() {
		Worker(simpleMap, nil)
		done <- true
	}()

	// Wait for worker completion
	<-done

	// Verify intermediate file mr-0-0 exists and has correct content
	intermediateFile := "mr-0-0"
	defer os.Remove(intermediateFile)

	// Check file exists
	if _, err := os.Stat(intermediateFile); os.IsNotExist(err) {
		t.Fatalf("Expected intermediate file %s to exist", intermediateFile)
	}

	// Read file and verify JSON content
	file, err := os.Open(intermediateFile)
	if err != nil {
		t.Fatalf("Error opening intermediate file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var keyValues []KeyValue

	// Read line by line with json.Decoder
	for decoder.More() {
		var kv KeyValue
		if err := decoder.Decode(&kv); err != nil {
			t.Fatalf("Error decoding JSON from intermediate file: %v", err)
		}
		keyValues = append(keyValues, kv)
	}

	// Verify expected key-value pairs are present
	// For "hello world hello": expect to find "hello":"1" twice, "world":"1" once
	expectedKeys := map[string]int{
		"hello": 2,
		"world": 1,
	}

	actualKeys := make(map[string]int)
	for _, kv := range keyValues {
		if kv.Value != "1" {
			t.Fatalf("Expected all values to be '1', but got '%s' for key '%s'", kv.Value, kv.Key)
		}
		actualKeys[kv.Key]++
	}

	for expectedKey, expectedCount := range expectedKeys {
		if actualCount, exists := actualKeys[expectedKey]; !exists {
			t.Fatalf("Expected key '%s' not found in intermediate file", expectedKey)
		} else if actualCount != expectedCount {
			t.Fatalf("Expected key '%s' to appear %d times, but found %d times", expectedKey, expectedCount, actualCount)
		}
	}

	// Verify no unexpected keys
	for actualKey := range actualKeys {
		if _, expected := expectedKeys[actualKey]; !expected {
			t.Fatalf("Unexpected key '%s' found in intermediate file", actualKey)
		}
	}
}
