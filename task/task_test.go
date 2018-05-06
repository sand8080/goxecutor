package task_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/sand8080/goxecutor/task"
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
	nilReq := NewTask("P", nil, nil, dumpDoFunc, nil)
	assert.Equal(t, StatusNew, nilReq.Status, "Expected task status: %v, got: %v",
		StatusNew, nilReq.Status)
	assert.NotNil(t, nilReq.RequiredFor, "Added child map doesn't initialized")
	assert.Nil(t, nilReq.WaitingResult, "Waiting channel must be nil if requires empty or nil")

	emptyReq := NewTask("P", []ID{}, nil, dumpDoFunc, nil)
	assert.Equal(t, StatusNew, emptyReq.Status, "Expected task status: %v, got: %v",
		StatusNew, emptyReq.Status)
	assert.NotNil(t, emptyReq.RequiredFor, "Added child map doesn't initialized")
	assert.Nil(t, emptyReq.WaitingResult, "Waiting channel must be nil if requires empty or nil")

	withReq := NewTask("WithReq", []ID{"Req1", "Req2"}, nil, dumpDoFunc, nil)
	assert.True(t, withReq.Requires["Req1"])
	assert.True(t, withReq.Requires["Req2"])
	assert.NotNil(t, withReq.WaitingResult)
}

func TestTask_AddChild(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc, nil)
	chOne := NewTask("ChOne", []ID{"P"}, nil, dumpDoFunc, nil)
	err := parent.AddChild(chOne)
	assert.NoError(t, err)

	chTwo := NewTask("ChTwo", []ID{"P"}, nil, dumpDoFunc, nil)
	err = parent.AddChild(chTwo)
	assert.NoError(t, err)

	// Checking count of added child
	assert.True(t, parent.RequiredFor["ChOne"])
	assert.True(t, parent.RequiredFor["ChTwo"])

	// Checking channels are registered in parent
	// Channel type cast to chan<-
	var woCh chan<- Result = chOne.WaitingResult
	assert.Contains(t, parent.NotifyResult, woCh)

	woCh = chTwo.WaitingResult
	assert.Contains(t, parent.NotifyResult, woCh)
}

func TestTask_AddChild_NotChild(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc, nil)
	chOne := NewTask("ChOne", nil, nil, dumpDoFunc, nil)
	assert.Error(t, ErrNotChild, parent.AddChild(chOne))
}

func TestTask_AddChild_NotAllRequirements(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc, nil)
	ch := NewTask("Ch", []ID{"P", "Q", "R"}, nil, dumpDoFunc, nil)
	assert.NoError(t, parent.AddChild(ch))
}

func TestTask_AddChild_AlreadyAdded(t *testing.T) {
	parent := NewTask("P", nil, nil, dumpDoFunc, nil)
	chOne := NewTask("ChOne", []ID{"P"}, nil, dumpDoFunc, nil)
	assert.NoError(t, parent.AddChild(chOne))
	assert.Error(t, ErrChildAlreadyAdded, parent.AddChild(chOne))
}

func TestDo(t *testing.T) {
	parent := NewTask("P", nil, "P", dumpDoFunc, nil)
	chOne := NewTask("ChOne", []ID{"P"}, "ChOne", dumpDoFunc, nil)
	assert.NoError(t, parent.AddChild(chOne))
	chTwo := NewTask("ChTwo", []ID{"P"}, "ChTwo", dumpDoFunc, nil)
	assert.NoError(t, parent.AddChild(chTwo))
	chLeaf := NewTask("ChLeaf", []ID{"ChOne", "ChTwo"}, "ChLeaf", dumpDoFunc, nil)
	assert.NoError(t, chOne.AddChild(chLeaf))
	assert.NoError(t, chTwo.AddChild(chLeaf))

	ctx, cancelFunc := context.WithCancel(context.Background())
	storage := newMockedStorage()
	go Do(ctx, cancelFunc, parent, storage)
	go Do(ctx, cancelFunc, chOne, storage)
	go Do(ctx, cancelFunc, chTwo, storage)

	// Waiting Do result for data consistency guarantee
	assert.NoError(t, Do(ctx, cancelFunc, chLeaf, storage))

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
			"Task %v status expected: %v, actual: %v", check.task.ID,
			check.statusExp, check.task.Status)
	}
}

func TestDo_Cancellation(t *testing.T) {
	parent := NewTask("P", nil, "P", dumpDoFunc, nil)
	// Task child will be failed
	errFunc := func(ctx context.Context, p interface{}) (interface{}, error) { return nil, errors.New("stop") }
	child := NewTask("Ch", []ID{"P"}, "Ch", errFunc, nil)
	assert.NoError(t, parent.AddChild(child))
	// Task leaf shouldn't be executed
	leaf := NewTask("Leaf", []ID{"Ch"}, "Leaf", dumpDoFunc, nil)
	assert.NoError(t, child.AddChild(leaf))

	ctx, cancelFunc := context.WithCancel(context.Background())
	storage := newMockedStorage()
	go Do(ctx, cancelFunc, parent, storage)
	go Do(ctx, cancelFunc, child, storage)

	// Waiting Do result for data consistency guarantee
	assert.NoError(t, Do(ctx, cancelFunc, leaf, storage))

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
			"Task %v status expected: %v, actual: %v", check.task.ID,
			check.statusExp, check.task.Status)
	}
}

func TestDo_DataPipeline(t *testing.T) {
	// Parent returns payload
	pOneID, pOneValue := ID("POneID"), "POne result"
	pOneFunc := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return pOneValue, nil
	}
	pOne := NewTask(pOneID, nil, nil, pOneFunc, nil)

	// Parent returns payload
	pTwoID, pTwoValue := ID("PTwoID"), "PTwo result"
	pTwoFunc := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return pTwoValue, nil
	}
	pTwo := NewTask(pTwoID, nil, nil, pTwoFunc, nil)

	// Parent returns no payload
	pThreeID := ID("PThreeID")
	pThree := NewTask(pThreeID, nil, nil, dumpDoFunc, nil)

	// Child requires all parents
	childFunc := func(ctx context.Context, payload interface{}) (interface{}, error) {
		log.Debugf("ctx: %v", ctx)
		assert.Equal(t, pOneValue, ctx.Value(pOneID))
		assert.Equal(t, pTwoValue, ctx.Value(pTwoID))
		assert.Nil(t, ctx.Value(pThreeID))
		return nil, nil
	}
	child := NewTask("Child", []ID{pOneID, pTwoID, pThreeID}, nil, childFunc, nil)

	// Adding child
	assert.NoError(t, pOne.AddChild(child))
	assert.NoError(t, pTwo.AddChild(child))
	assert.NoError(t, pThree.AddChild(child))

	ctx, cancelFunc := context.WithCancel(context.Background())
	storage := newMockedStorage()
	go Do(ctx, cancelFunc, pOne, storage)
	go Do(ctx, cancelFunc, pTwo, storage)
	go Do(ctx, cancelFunc, pThree, storage)
	assert.NoError(t, Do(ctx, cancelFunc, child, storage))
}

func Test_DoRespSaved(t *testing.T) {
	checks := []struct {
		do       DoFunc
		expected []byte
	}{
		{dumpDoFunc, nil},
		{func(context.Context, interface{}) (interface{}, error) {
			return "sыь", nil
		}, []byte("\"sыь\"")},
		{func(context.Context, interface{}) (interface{}, error) {
			return map[string]int{"a": 1}, nil
		}, []byte("{\"a\":1}")},
		{func(context.Context, interface{}) (interface{}, error) {
			return map[string][]int{"a": {1, 2}}, nil
		}, []byte("{\"a\":[1,2]}")},
	}

	storage := newMockedStorage()
	for _, check := range checks {
		task := NewTask("T", nil, nil, check.do, nil)
		assert.NoError(t, Do(nil, nil, task, storage))
		assert.Equal(t, check.expected, task.DoResult, "Expected: %s, actual: %s",
			check.expected, task.DoResult)
	}
}
