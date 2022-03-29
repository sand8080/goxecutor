package example

import (
	"encoding/json"
	"testing"

	"github.com/sand8080/goxecutor/internal"

	"github.com/stretchr/testify/assert"
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
	g := internal.NewGraph("GetBalance")
	authTask := NewAuthTask("auth", authServer.URL, "l", "p")
	_ = g.Add(authTask)
	balanceTask := NewBalanceTask("balance", balanceServer.URL, authTask.ID)
	_ = g.Add(balanceTask)

	// Checking graph
	assert.NoError(t, g.Check())

	// Executing graph
	status, err := g.Exec(internal.PolicyIgnoreError, NewMockedStorage())
	assert.Equal(t, internal.StatusSuccess, status)
	assert.NoError(t, err)

	// Checking tasks statuses
	var actToken string
	assert.NoError(t, json.Unmarshal(authTask.DoResult, &actToken))
	assert.Equal(t, token, actToken)

	var actBalance int
	assert.NoError(t, json.Unmarshal(balanceTask.DoResult, &actBalance))
	assert.Equal(t, balance, actBalance)
}
