package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/NerRox/payment-app/internal/app"
	"github.com/NerRox/payment-app/internal/database"
)

func TestBalanceHandler(t *testing.T) {
	conn := app.DBConnection{Connection: database.MustConnectPostgres()}
	defer conn.Connection.Close()

	http.HandleFunc("/balance", conn.BalanceHandler)
	http.HandleFunc("/transfer", conn.TransferHandler)
	http.HandleFunc("/withdraw", conn.WithdrawHandler)
	http.HandleFunc("/enroll", conn.EnrollHandler)
	if err := http.ListenAndServe("0.0.0.0:"+os.Getenv("APP_PORT"), nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/balance?id=11235", nil)
	w := httptest.NewRecorder()
	conn.BalanceHandler(w, req)
	res := w.Result()
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if string(data) != "2500" {
		t.Errorf("expected ABC got %v", string(data))
	}
}
