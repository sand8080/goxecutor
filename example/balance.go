package example

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/sand8080/goxecutor/task"
)

// BalanceReq example balance request
type BalanceReq struct {
	url      string
	tokenKey task.ID
}

// BalanceResp example balance response
type BalanceResp struct {
	Status  string `json:"status"`
	Balance int    `json:"balance"`
}

// BalanceFunc example implementation of business logic
func BalanceFunc(ctx context.Context, payload interface{}) (interface{}, error) {
	balance, ok := payload.(BalanceReq)
	if !ok {
		return nil, errors.New("cast to BalanceReq failed")
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", balance.url, nil)
	if err != nil {
		return nil, err
	}

	token, ok := ctx.Value(balance.tokenKey).(string)
	if !ok {
		return nil, errors.New("can't extract token value")
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	balanceResp := BalanceResp{}
	err = json.NewDecoder(resp.Body).Decode(&balanceResp)
	if err != nil {
		return nil, err
	}
	if balanceResp.Status != "ok" {
		return nil, errors.New("balance failed")
	}

	return balanceResp.Balance, nil
}

// NewBalanceTask creates balance fetching task
func NewBalanceTask(ID task.ID, url string, tokenKey task.ID) *task.Task {
	payload := BalanceReq{url, tokenKey}
	return task.NewTask(ID, []task.ID{tokenKey}, payload, BalanceFunc, nil)
}

// BalanceServer implements test auth server
func BalanceServer(token string, balance int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		actToken := r.Header.Get("X-Auth-Token")

		var resp BalanceResp
		if actToken == token {
			resp = BalanceResp{Status: "ok", Balance: balance}
		} else {
			resp = BalanceResp{Status: "failed"}
		}

		data, err := json.Marshal(resp)
		if err != nil {
			writeError(w, err)
			return
		}
		w.Write(data)
	}))
}
