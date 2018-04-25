package task

import (
	"context"
	"errors"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

type payloadSyncExec struct {
	taskID ID
	wg     *sync.WaitGroup
}

func syncedExedFunc(payload interface{}) error {
	log.Debugf("syncedExedFunc called with: %v", payload)
	p, ok := payload.(payloadSyncExec)
	if !ok {
		return errors.New("syncedExedFunc failed on payload cast to payloadSyncExec")
	}
	defer p.wg.Done()
	return nil
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

func TestTask_AddChild_NoMaxCount(t *testing.T) {
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

func Test_Exec(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(4)

	parent := NewTask("P", nil, payloadSyncExec{taskID: "P", wg: &wg}, syncedExedFunc)
	chOne := NewTask("ChOne", []ID{"P"}, payloadSyncExec{taskID: "ChOne", wg: &wg}, syncedExedFunc)
	assert.NoError(t, parent.AddChild(*chOne))
	chTwo := NewTask("ChTwo", []ID{"P"}, payloadSyncExec{taskID: "ChTwo", wg: &wg}, syncedExedFunc)
	assert.NoError(t, parent.AddChild(*chTwo))
	chLeaf := NewTask("ChLeaf", []ID{"ChOne", "ChTwo"}, payloadSyncExec{taskID: "ChLeaf", wg: &wg}, syncedExedFunc)
	assert.NoError(t, chOne.AddChild(*chLeaf))
	assert.NoError(t, chTwo.AddChild(*chLeaf))

	ctx, cancelFunc := context.WithCancel(context.Background())
	go Exec(ctx, cancelFunc, parent)
	go Exec(ctx, cancelFunc, chOne)
	go Exec(ctx, cancelFunc, chTwo)

	// Waiting Exec result for data consistency guarantee
	assert.NoError(t, Exec(ctx, cancelFunc, chLeaf))
	wg.Wait()

	assert.Equal(t, parent.Status, StatusReady)
	assert.Equal(t, chOne.Status, StatusReady)
	assert.Equal(t, chTwo.Status, StatusReady)
	assert.Equal(t, chLeaf.Status, StatusReady)
}
