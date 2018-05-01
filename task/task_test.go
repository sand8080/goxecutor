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

func dumpDoFunc(ctx context.Context, payload interface{}) (interface{}, error) {
	log.Debugf("Dump exec function is called with payload: %#v", payload)
	return nil, nil
}

func TestNewTask(t *testing.T) {
	nilReq := NewTask("P", nil, nil, dumpDoFunc)
	assert.Equal(t, StatusNew, nilReq.Status, "Expected task status: %v, got: %v", StatusNew, nilReq.Status)
	assert.NotNil(t, nilReq.RequiredFor, "Added child map doesn't initialized")
	assert.Nil(t, nilReq.waitingResult, "Waiting channel must be nil if requires empty or nil")

	emptyReq := NewTask("P", []ID{}, nil, dumpDoFunc)
	assert.Equal(t, StatusNew, emptyReq.Status, "Expected task status: %v, got: %v", StatusNew, emptyReq.Status)
	assert.NotNil(t, emptyReq.RequiredFor, "Added child map doesn't initialized")
	assert.Nil(t, emptyReq.waitingResult, "Waiting channel must be nil if requires empty or nil")

	withReq := NewTask("WithReq", []ID{"Req1", "Req2"}, nil, dumpDoFunc)
	assert.True(t, withReq.Requires["Req1"])
	assert.True(t, withReq.Requires["Req2"])
	assert.NotNil(t, withReq.waitingResult)
}

func TestTask_AddChild(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc)
	chOne := NewTask("ChOne", []ID{"P"}, nil, dumpDoFunc)
	err := parent.AddChild(chOne)
	assert.NoError(t, err)

	chTwo := NewTask("ChTwo", []ID{"P"}, nil, dumpDoFunc)
	err = parent.AddChild(chTwo)
	assert.NoError(t, err)

	// Checking count of added child
	assert.True(t, parent.RequiredFor["ChOne"])
	assert.True(t, parent.RequiredFor["ChTwo"])

	// Checking channels are registered in parent
	// Channel type cast to chan<-
	var woCh chan<- taskResult = chOne.waitingResult
	assert.Contains(t, parent.notifyResult, woCh)

	woCh = chTwo.waitingResult
	assert.Contains(t, parent.notifyResult, woCh)
}

func TestTask_AddChild_NotChild(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc)
	chOne := NewTask("ChOne", nil, nil, dumpDoFunc)
	assert.Error(t, ErrNotChild, parent.AddChild(chOne))
}

func TestTask_AddChild_NotAllRequirements(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc)
	ch := NewTask("Ch", []ID{"P", "Q", "R"}, nil, dumpDoFunc)
	assert.NoError(t, parent.AddChild(ch))
}

func TestTask_AddChild_AlreadyAdded(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc)
	chOne := NewTask("ChOne", []ID{"P"}, nil, dumpDoFunc)
	assert.NoError(t, parent.AddChild(chOne))
	assert.Error(t, ErrChildAlreadyAdded, parent.AddChild(chOne))
}

func TestExec(t *testing.T) {
	parent := NewTask("P", nil, "P", dumpDoFunc)
	chOne := NewTask("ChOne", []ID{"P"}, "ChOne", dumpDoFunc)
	assert.NoError(t, parent.AddChild(chOne))
	chTwo := NewTask("ChTwo", []ID{"P"}, "ChTwo", dumpDoFunc)
	assert.NoError(t, parent.AddChild(chTwo))
	chLeaf := NewTask("ChLeaf", []ID{"ChOne", "ChTwo"}, "ChLeaf", dumpDoFunc)
	assert.NoError(t, chOne.AddChild(chLeaf))
	assert.NoError(t, chTwo.AddChild(chLeaf))

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
	parent := NewTask("P", nil, "P", dumpDoFunc)
	// Task child will be failed
	errFunc := func(ctx context.Context, p interface{}) (interface{}, error) { return nil, errors.New("stop") }
	child := NewTask("Ch", []ID{"P"}, "Ch", errFunc)
	assert.NoError(t, parent.AddChild(child))
	// Task leaf shouldn't be executed
	leaf := NewTask("Leaf", []ID{"Ch"}, "Leaf", dumpDoFunc)
	assert.NoError(t, child.AddChild(leaf))

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

func TestExec_DataPipeline(t *testing.T) {
	// Parent returns payload
	pOneID, pOneValue := ID("POneID"), "POne result"
	pOneFunc := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return pOneValue, nil
	}
	pOne := NewTask(pOneID, nil, nil, pOneFunc)

	// Parent returns payload
	pTwoID, pTwoValue := ID("PTwoID"), "PTwo result"
	pTwoFunc := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return pTwoValue, nil
	}
	pTwo := NewTask(pTwoID, nil, nil, pTwoFunc)

	// Parent returns no payload
	pThreeID := ID("PThreeID")
	pThree := NewTask(pThreeID, nil, nil, dumpDoFunc)

	// Child requires all parents
	childFunc := func(ctx context.Context, payload interface{}) (interface{}, error) {
		log.Debugf("ctx: %v", ctx)
		assert.Equal(t, pOneValue, ctx.Value(pOneID))
		assert.Equal(t, pTwoValue, ctx.Value(pTwoID))
		assert.Nil(t, ctx.Value(pThreeID))
		return nil, nil
	}
	child := NewTask("Child", []ID{pOneID, pTwoID, pThreeID}, nil, childFunc)

	// Adding child
	assert.NoError(t, pOne.AddChild(child))
	assert.NoError(t, pTwo.AddChild(child))
	assert.NoError(t, pThree.AddChild(child))

	ctx, cancelFunc := context.WithCancel(context.Background())
	go Exec(ctx, cancelFunc, pOne)
	go Exec(ctx, cancelFunc, pTwo)
	go Exec(ctx, cancelFunc, pThree)
	assert.NoError(t, Exec(ctx, cancelFunc, child))
}

// Move to Job
//func TestExec_NotAllParents(t *testing.T) {
//	parent := NewTask("P", nil, "P", dumpDoFunc)
//	child := NewTask("P", []ID{"P", "PPP"}, "P", dumpDoFunc)
//	assert.NoError(t, parent.AddChild(*child))
//
//	ctx, cancelFunc := context.WithCancel(context.Background())
//	go Exec(ctx, cancelFunc, parent)
//	err := Exec(ctx, cancelFunc, child)
//
//	assert.Equal(t, err, "")
//}
