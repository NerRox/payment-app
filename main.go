package main

import (
	"log"
	"net/http"
	"os"

	"github.com/NerRox/payment-app/internal/app"
	"github.com/NerRox/payment-app/internal/database"
)

func main() {
	conn := app.DBConnection{Connection: database.MustConnectPostgres()}
	defer conn.Connection.Close()

	http.HandleFunc("/balance", conn.BalanceHandler)
	http.HandleFunc("/transfer", conn.TransferHandler)
	http.HandleFunc("/withdraw", conn.WithdrawHandler)
	http.HandleFunc("/enroll", conn.EnrollHandler)
	if err := http.ListenAndServe("0.0.0.0:"+os.Getenv("APP_PORT"), nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
