package example

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sand8080/goxecutor/graph"
)

func Test_GetBalance(t *testing.T) {
	// auth -> (token) -> balance
	token := "token-123-321"
	authServer := AuthServer("l", "p", token)
	defer authServer.Close()
	balance := 42
	balanceServer := BalanceServer(token, balance)
	defer balanceServer.Close()

	// Constructing tasks graph
	g := graph.NewGraph("GetBalance")
	authTask := NewAuthTask("auth", authServer.URL, "l", "p")
	g.Add(authTask)
	balanceTask := NewBalanceTask("balance", balanceServer.URL, authTask.ID)
	g.Add(balanceTask)

	// Checking graph
	assert.NoError(t, g.Check())

	// Executing graph
	status, err := g.Exec(graph.PolicyIgnoreError, NewMockedStorage())
	assert.Equal(t, graph.StatusSuccess, status)
	assert.NoError(t, err)

	// Checking tasks statuses
	var actToken string
	assert.NoError(t, json.Unmarshal(authTask.DoResult, &actToken))
	assert.Equal(t, token, actToken)

	var actBalance int
	assert.NoError(t, json.Unmarshal(balanceTask.DoResult, &actBalance))
	assert.Equal(t, balance, actBalance)
}
