package example

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sand8080/goxecutor/task"
	"io/ioutil"
	"net/http/httptest"
)

// AuthReq auth example request
type AuthReq struct {
	url      string
	Login    string `json:"login"`
	Password string `json:"password"`
}

// AuthResp auth example response
type AuthResp struct {
	Status string `json:"status"`
	Token  string `json:"token"`
}

// AuthFunc example implementation of business logic
func AuthFunc(ctx context.Context, payload interface{}) (interface{}, error) {
	auth, ok := payload.(AuthReq)
	if !ok {
		return nil, errors.New("cast to AuthReq failed")
	}
	raw, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(auth.url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	authResp := AuthResp{}
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	if err != nil {
		return nil, err
	}
	if authResp.Status != "ok" {
		return nil, errors.New("auth failed")
	}

	return authResp.Token, nil
}

// NewAuthTask creates auth task
func NewAuthTask(ID task.ID, url, login, password string) *task.Task {
	payload := AuthReq{url, login, password}
	return task.NewTask(ID, nil, payload, AuthFunc, nil)
}

// AuthServer implements test auth server
func AuthServer(login, password, token string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(w, err)
			return
		}

		req := AuthReq{}
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, err)
			return
		}

		var resp AuthResp
		if req.Login == login && req.Password == password {
			resp = AuthResp{Status: "ok", Token: token}
		} else {
			resp = AuthResp{Status: "failed", Token: ""}
		}

		data, err := json.Marshal(resp)
		if err != nil {
			writeError(w, err)
			return
		}
		w.Write(data)
	}))
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
