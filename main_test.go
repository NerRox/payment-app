package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBalanceHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/balance?id=11235", nil)
	w := httptest.NewRecorder()
	balanceHandler(w, req)
	res := w.Result()
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if string(data) != "2500" {
		t.Errorf("expected ABC got %v", string(data))
	}
}
