package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
)

type BalanceAnswer struct {
	UserID      int `json:"userId"`
	UserBalance int `json:"userBalance"`
}

// Зачисление денег на баланс пользователя
func enrollHandler(w http.ResponseWriter, r *http.Request) {
	enrollActionId, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	enrollActionAmount, err := strconv.Atoi(r.URL.Query().Get("amount"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	urlExample := "postgres://username:password@localhost:5432/users"
	conn, err := pgx.Connect(context.Background(), urlExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// Получаем текущий баланс пользователя и записываем его в переменную
	var currentUserAmount int = 0
	err = conn.QueryRow(context.Background(), "select balance from users where userid="+strconv.Itoa(enrollActionId)).Scan(&currentUserAmount)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	// Записываем новый баланс пользователя в таблицу
	userId := 0
	err = conn.QueryRow(context.Background(), "insert into users(userid, balance) values($1, $2) returning userid", enrollActionId, enrollActionAmount+currentUserAmount).Scan(&userId)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	// Формируем ответ и возвращаем его
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	balanceAnswer := BalanceAnswer{enrollActionId, enrollActionAmount + currentUserAmount}

	jsonResp, err := json.Marshal(balanceAnswer)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}

// Функция по списанию средств
func withdrawHandler(w http.ResponseWriter, r *http.Request) {
	withdrawActionId, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	withdrawActionAmount, err := strconv.Atoi(r.URL.Query().Get("amount"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	urlExample := "postgres://username:password@localhost:5432/users"
	conn, err := pgx.Connect(context.Background(), urlExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// Получаем текущий баланс пользователя и записываем его в переменную
	var currentUserAmount int = 0
	err = conn.QueryRow(context.Background(), "select balance from users where userid="+strconv.Itoa(withdrawActionId)).Scan(&currentUserAmount)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	if currentUserAmount < withdrawActionAmount {
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "application/json")
		jsonResp, err := json.Marshal("Unsufficient funds")
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
	} else {
		var calculatedAmount int = currentUserAmount - withdrawActionAmount
		var userId string

		err = conn.QueryRow(context.Background(), "insert into users(userid, balance) values($1, $2) returning userid", withdrawActionId, calculatedAmount).Scan(&userId)

		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
			os.Exit(1)
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		balanceAnswer := BalanceAnswer{withdrawActionId, calculatedAmount}

		jsonResp, err := json.Marshal(balanceAnswer)
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)

	}
}

// Функция, которая переводит деньги от одного пользователя другому
func transferHandler(w http.ResponseWriter, r *http.Request) {
	senderId, err := strconv.Atoi(r.URL.Query().Get("senderId"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	receiverId, err := strconv.Atoi(r.URL.Query().Get("receiverId"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	transferAmount, err := strconv.Atoi(r.URL.Query().Get("amount"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	urlExample := "postgres://username:password@localhost:5432/users"
	conn, err := pgx.Connect(context.Background(), urlExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// Получаем текущий баланс пользователя-отправителя и записываем его в переменную
	var currentSenderAmount int
	err = conn.QueryRow(context.Background(), "select balance from users where userid="+strconv.Itoa(senderId)).Scan(&currentSenderAmount)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	if transferAmount > currentSenderAmount {
		w.WriteHeader(http.StatusForbidden)
		w.Header().Set("Content-Type", "application/json")
		jsonResp, err := json.Marshal("Unsufficient funds")
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
	} else {
		// Получаем текущий баланс пользователя-получателя и записываем его в переменную
		var currentReceiverAmount int
		err = conn.QueryRow(context.Background(), "select balance from users where userid="+strconv.Itoa(receiverId)).Scan(&currentReceiverAmount)

		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
			os.Exit(1)
		}

		// Выставляем новый баланс пользователя-отправителя
		senderNewAmount := currentSenderAmount - transferAmount
		var userId error
		err = conn.QueryRow(context.Background(), "insert into users(userid, balance) values($1, $2) returning userid", senderId, senderNewAmount).Scan(&userId)

		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
			os.Exit(1)
		}

		// Выставляем новый баланс пользователя-получателя
		receiverNewAmount := currentReceiverAmount + transferAmount
		err = conn.QueryRow(context.Background(), "insert into users(userid, balance) values($1, $2) returning userid", receiverId, receiverNewAmount).Scan(&userId)

		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
			os.Exit(1)
		}

		// Формируем ответ и возвращаем его
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		balanceAnswer := BalanceAnswer{receiverId, receiverNewAmount}

		jsonResp, err := json.Marshal(balanceAnswer)
		if err != nil {
			log.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)

	}
}

// Функция, которая возвращает баланс пользователя
func balanceHandler(w http.ResponseWriter, r *http.Request) {
	balanceActionId, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	urlExample := "postgres://username:password@localhost:5432/users"
	conn, err := pgx.Connect(context.Background(), urlExample)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	// Получаем текущий баланс пользователя и записываем его в переменную
	var currentUserAmount int = 0
	err = conn.QueryRow(context.Background(), "select balance from users where userid="+strconv.Itoa(balanceActionId)).Scan(&currentUserAmount)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	// Формируем ответ и возвращаем его
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	balanceAnswer := BalanceAnswer{balanceActionId, currentUserAmount}

	jsonResp, err := json.Marshal(balanceAnswer)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	w.Write(jsonResp)
}

func main() {
	http.HandleFunc("/balance", balanceHandler)
	http.HandleFunc("/transfer", transferHandler)
	http.HandleFunc("/withdraw", withdrawHandler)
	http.HandleFunc("/enroll", enrollHandler)
	http.ListenAndServe("localhost:8080", nil)
}
