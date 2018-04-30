package task

import (
	"context"
	"testing"

	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func dumpExecFunc(payload interface{}) error {
	log.Debugf("Dump exec function is called with payload: %#v", payload)
	return nil
}

func TestNewTask(t *testing.T) {
	nilReq := NewTask("P", nil, nil, dumpExecFunc)
	assert.Equal(t, StatusNew, nilReq.Status, "Expected task status: %v, got: %v", StatusNew, nilReq.Status)
	assert.NotNil(t, nilReq.RequiredFor, "Added child map doesn't initialized")
	assert.Nil(t, nilReq.waitingID, "Waiting channel must be nil if requires empty or nil")

	emptyReq := NewTask("P", []ID{}, nil, dumpExecFunc)
	assert.Equal(t, StatusNew, emptyReq.Status, "Expected task status: %v, got: %v", StatusNew, emptyReq.Status)
	assert.NotNil(t, emptyReq.RequiredFor, "Added child map doesn't initialized")
	assert.Nil(t, emptyReq.waitingID, "Waiting channel must be nil if requires empty or nil")

	withReq := NewTask("WithReq", []ID{"Req1", "Req2"}, nil, dumpExecFunc)
	assert.True(t, withReq.Requires["Req1"])
	assert.True(t, withReq.Requires["Req2"])
	assert.NotNil(t, withReq.waitingID)
}

func TestTask_AddChild(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpExecFunc)
	chOne := *NewTask("ChOne", []ID{"P"}, nil, dumpExecFunc)
	err := parent.AddChild(chOne)
	assert.NoError(t, err)

	chTwo := *NewTask("ChTwo", []ID{"P"}, nil, dumpExecFunc)
	err = parent.AddChild(chTwo)
	assert.NoError(t, err)

	// Checking count of added child
	assert.True(t, parent.RequiredFor["ChOne"])
	assert.True(t, parent.RequiredFor["ChTwo"])

	// Checking channels are registered in parent
	// Channel type cast to chan<-
	var woCh chan<- ID = chOne.waitingID
	assert.Contains(t, parent.notifyIDs, woCh)

	woCh = chTwo.waitingID
	assert.Contains(t, parent.notifyIDs, woCh)
}

func TestTask_AddChild_NotChild(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpExecFunc)
	chOne := *NewTask("ChOne", nil, nil, dumpExecFunc)
	assert.Error(t, ErrNotChild, parent.AddChild(chOne))
}

func TestTask_AddChild_NotAllRequirements(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpExecFunc)
	ch := *NewTask("Ch", []ID{"P", "Q", "R"}, nil, dumpExecFunc)
	assert.NoError(t, parent.AddChild(ch))
}

func TestTask_AddChild_AlreadyAdded(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpExecFunc)
	chOne := *NewTask("ChOne", []ID{"P"}, nil, dumpExecFunc)
	assert.NoError(t, parent.AddChild(chOne))
	assert.Error(t, ErrChildAlreadyAdded, parent.AddChild(chOne))
}

func TestExec(t *testing.T) {
	parent := NewTask("P", nil, "P", dumpExecFunc)
	chOne := NewTask("ChOne", []ID{"P"}, "ChOne", dumpExecFunc)
	assert.NoError(t, parent.AddChild(*chOne))
	chTwo := NewTask("ChTwo", []ID{"P"}, "ChTwo", dumpExecFunc)
	assert.NoError(t, parent.AddChild(*chTwo))
	chLeaf := NewTask("ChLeaf", []ID{"ChOne", "ChTwo"}, "ChLeaf", dumpExecFunc)
	assert.NoError(t, chOne.AddChild(*chLeaf))
	assert.NoError(t, chTwo.AddChild(*chLeaf))

	ctx, cancelFunc := context.WithCancel(context.Background())
	go Exec(ctx, cancelFunc, parent)
	go Exec(ctx, cancelFunc, chOne)
	go Exec(ctx, cancelFunc, chTwo)

	// Waiting Exec result for data consistency guarantee
	assert.NoError(t, Exec(ctx, cancelFunc, chLeaf))

	checks := []struct {
		task      *Task
		statusExp Status
	}{
		{parent, StatusReady},
		{chOne, StatusReady},
		{chTwo, StatusReady},
		{chLeaf, StatusReady},
	}

	for _, check := range checks {
		assert.Equal(t, check.statusExp, check.task.Status,
			"Task %v status expected: %v, actual: %v", check.task.ID, check.statusExp, check.task.Status)
	}
}

func TestExec_Cancellation(t *testing.T) {
	parent := NewTask("P", nil, "P", dumpExecFunc)
	// Task child will be failed
	errFunc := func(p interface{}) error { return errors.New("stop") }
	child := NewTask("Ch", []ID{"P"}, "Ch", errFunc)
	assert.NoError(t, parent.AddChild(*child))
	// Task leaf shouldn't be executed
	leaf := NewTask("Leaf", []ID{"Ch"}, "Leaf", dumpExecFunc)
	assert.NoError(t, child.AddChild(*leaf))

	ctx, cancelFunc := context.WithCancel(context.Background())
	go Exec(ctx, cancelFunc, parent)
	go Exec(ctx, cancelFunc, child)

	// Waiting Exec result for data consistency guarantee
	assert.NoError(t, Exec(ctx, cancelFunc, leaf))

	checks := []struct {
		task      *Task
		statusExp Status
	}{
		{parent, StatusReady},
		{child, StatusError},
		{leaf, StatusCancelled},
	}

	for _, check := range checks {
		assert.Equal(t, check.statusExp, check.task.Status,
			"Task %v status expected: %v, actual: %v", check.task.ID, check.statusExp, check.task.Status)
	}
}

// Move to Job
//func TestExec_NotAllParents(t *testing.T) {
//	parent := NewTask("P", nil, "P", dumpExecFunc)
//	child := NewTask("P", []ID{"P", "PPP"}, "P", dumpExecFunc)
//	assert.NoError(t, parent.AddChild(*child))
//
//	ctx, cancelFunc := context.WithCancel(context.Background())
//	go Exec(ctx, cancelFunc, parent)
//	err := Exec(ctx, cancelFunc, child)
//
//	assert.Equal(t, err, "")
//}
