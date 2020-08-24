package main

import (
	"testing"
)

func TestIsTestID(t *testing.T) {
	assertTrue(t, isTaskARN("arn:aws:ecs:us-east-1:1111111111:task/dev-cluster/0b4b2b4daf475ee0bf19157238902649"))
	assertTrue(t, isTaskARN("arn:aws:ecs:us-east-1:1111111111:task/Staging/0b4b2b4daf475ee0bf19157238902649"))
	assertTrue(t, isTaskARN("arn:aws:ecs:us-west-2:4817267453:task/bfbf861b-7f10-4dfb-b344-32169dc3e55c"))

	assertFalse(t, isTaskARN("bad-prefix/0b4b2b4daf475ee0bf19157238902649"))
	assertFalse(t, isTaskARN("arn:aws:ecs:us-east-1:1111111111:task/1234/0b4b2b4daf475ee0bf19157238902649"))

	assertTrue(t, isTaskID("0b4b2b4daf475ee0bf19157238902649"))
	assertTrue(t, isTaskID("bfbf861b-7f10-4dfb-b344-32169dc3e55c"))
}

func assertTrue(t *testing.T, v bool) {
	t.Helper()
	if !v {
		t.Errorf("Expected true, but was false")
	}
}

func assertFalse(t *testing.T, v bool) {
	t.Helper()
	if v {
		t.Errorf("Expected true, but was false")
	}
}
