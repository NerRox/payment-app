package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/NerRox/payment-app/internal/models"
)

type DBConnection struct {
	Connection *pgxpool.Pool
}

// Зачисление денег на баланс пользователя, создание пользователя при первом зачислении
func (c *DBConnection) EnrollHandler(w http.ResponseWriter, r *http.Request) {
	// Читаем Body из реквеста
	body, err := io.ReadAll(r.Body)

	// Выходим если боди не удалось распарсить
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read body: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Логируем тело запроса
	log.Println("Enroll request body is: " + string(body))

	// Пытаемся анмаршалить тело запроса
	var enrollRequestUnmarshalRes models.SingleUserRequest
	err = json.Unmarshal(body, &enrollRequestUnmarshalRes)

	// Если не выходит - логируем и выходим
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unmarshalling failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Сразу выходим, если значение баланса в запросе меньше нуля
	if enrollRequestUnmarshalRes.UserBalance < 0 {
		log.Printf("Balance in the query is less than a zero - %d!", enrollRequestUnmarshalRes.UserBalance)
		http.Error(w, "Balance in the query is less than a zero!", http.StatusForbidden)
		return
	}

	// Пытаемся получить текущий баланс пользователи и записать его в переменную
	var currentUserAmount int
	err = c.Connection.QueryRow(context.Background(), "SELECT balance FROM users WHERE userid=$1", enrollRequestUnmarshalRes.UserID).Scan(&currentUserAmount)

	// Если пользователь не найден - добавляем его в БД сразу с балансом, отвечаем и выходим
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = c.Connection.Exec(context.Background(), "INSERT INTO users(userid, balance) values($1, $2) RETURNING userid", enrollRequestUnmarshalRes.UserID, enrollRequestUnmarshalRes.UserBalance)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Insert new user to DB failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		log.Printf("User with ID %d added to DB with balance %d", enrollRequestUnmarshalRes.UserID, enrollRequestUnmarshalRes.UserBalance)

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")

		balanceAnswer := models.SingleUserRequest{UserID: enrollRequestUnmarshalRes.UserID, UserBalance: enrollRequestUnmarshalRes.UserBalance + currentUserAmount}

		jsonResp, err := json.Marshal(balanceAnswer)
		if err != nil {
			log.Printf("Error happened in JSON marshal. Err: %s", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Write(jsonResp)
		return
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Если ошибка не в отсутствии пользователя - падаем и выходим
		fmt.Fprintf(os.Stderr, "Get user balance QueryRow failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Если пользователь существовал - записываем новый пользователя в таблицу
	_, err = c.Connection.Exec(context.Background(), "UPDATE users SET balance=$1 WHERE userid=$2", enrollRequestUnmarshalRes.UserBalance+currentUserAmount, enrollRequestUnmarshalRes.UserID)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow err, failed to update user balance: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("User with ID %d updated in DB with balance %d", enrollRequestUnmarshalRes.UserID, enrollRequestUnmarshalRes.UserBalance+currentUserAmount)

	// Формируем ответ и возвращаем его
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	balanceAnswer := models.SingleUserRequest{UserID: enrollRequestUnmarshalRes.UserID, UserBalance: enrollRequestUnmarshalRes.UserBalance + currentUserAmount}

	jsonResp, err := json.Marshal(balanceAnswer)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Write(jsonResp)
}

// Функция по списанию средств
func (c *DBConnection) WithdrawHandler(w http.ResponseWriter, r *http.Request) {
	// Читаем Body из реквеста
	body, err := io.ReadAll(r.Body)

	// Выходим если что-то не так
	if err != nil {
		fmt.Fprintf(os.Stderr, "Reading body failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Печатаем тело запроса
	log.Printf("Withdraw request body is: %s", string(body))

	var WithdrawRequestUnmarshalRes models.SingleUserRequest
	err = json.Unmarshal(body, &WithdrawRequestUnmarshalRes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unmarshalling failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Если пытаемся списать число меньше 0 - вываливаемся
	if WithdrawRequestUnmarshalRes.UserBalance < 0 {
		log.Printf("You are trying to withdraw less than zero - %d.", WithdrawRequestUnmarshalRes.UserBalance)
		http.Error(w, "You are trying to withdraw less than zero.", http.StatusForbidden)
		return
	}

	// Получаем текущий баланс пользователя и записываем его в переменную
	var currentUserAmount int
	err = c.Connection.QueryRow(context.Background(), "SELECT balance FROM users WHERE userid=$1", WithdrawRequestUnmarshalRes.UserID).Scan(&currentUserAmount)

	// Если пользователя нет - выходим
	if errors.Is(err, pgx.ErrNoRows) {
		fmt.Fprintf(os.Stderr, "User not found, cannot withdraw: %v\n", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Если ошибка не в отсутствии пользователя - падаем и выходим
		fmt.Fprintf(os.Stderr, "Get user balance QueryRow failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Если пытаемся списать больше, чем есть у пользователя - нужно отдать ошибку
	if currentUserAmount < WithdrawRequestUnmarshalRes.UserBalance {
		var withdrawErrText string = fmt.Sprintf("User with ID " + strconv.Itoa(WithdrawRequestUnmarshalRes.UserID) + " balance is too low for this transaction: needs " + strconv.Itoa(WithdrawRequestUnmarshalRes.UserBalance) + ", has " + strconv.Itoa(currentUserAmount) + "!\n")
		fmt.Fprintln(os.Stderr, withdrawErrText)
		http.Error(w, withdrawErrText, http.StatusForbidden)
		return
	} else {
		var calculatedAmount int = currentUserAmount - WithdrawRequestUnmarshalRes.UserBalance

		_, err = c.Connection.Exec(context.Background(), "UPDATE users SET balance=$1 WHERE userid=$2", calculatedAmount, WithdrawRequestUnmarshalRes.UserID)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Не удалось записать списание средств: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		log.Printf("Withdraw successfull, user with ID %d now has balance %d", WithdrawRequestUnmarshalRes.UserID, calculatedAmount)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		balanceAnswer := models.SingleUserRequest{UserID: WithdrawRequestUnmarshalRes.UserID, UserBalance: calculatedAmount}

		jsonResp, err := json.Marshal(balanceAnswer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error happened in JSON marshalling EnrollHandler answer: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Write(jsonResp)

	}
}

// Функция, которая переводит деньги от одного пользователя другому
func (c *DBConnection) TransferHandler(w http.ResponseWriter, r *http.Request) {
	// Читаем Body из реквеста
	body, err := io.ReadAll(r.Body)

	// Сообщаем если что-то не так и выходим
	if err != nil {
		fmt.Fprintf(os.Stderr, "Reading transfer request body failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Печатаем тело запроса
	log.Printf("TransferHandler body is: %s", string(body))

	var TransferRequestUnmarshalRes models.TransferRequest
	err = json.Unmarshal(body, &TransferRequestUnmarshalRes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unmarshalling failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем текущий баланс пользователя-отправителя и записываем его в переменную
	var currentSenderAmount int
	err = c.Connection.QueryRow(context.Background(), "SELECT balance FROM users WHERE userid=$1", TransferRequestUnmarshalRes.SenderUserID).Scan(&currentSenderAmount)

	if errors.Is(err, pgx.ErrNoRows) {
		fmt.Fprintf(os.Stderr, "Sender user cannot be foubd: %v\n", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Если ошибка не в отсутствии пользователя - падаем и выходим
		fmt.Fprintf(os.Stderr, "Get sender user balance QueryRow failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("Sender user with ID %d pre-transfer balance is %d", TransferRequestUnmarshalRes.SenderUserID, currentSenderAmount)

	// Получаем текущий баланс пользователя-получателя и записываем его в переменную
	var currentReceiverAmount int
	err = c.Connection.QueryRow(context.Background(), "SELECT balance FROM users WHERE userid=$1", TransferRequestUnmarshalRes.ReceiverUserID).Scan(&currentReceiverAmount)

	if errors.Is(err, pgx.ErrNoRows) {
		fmt.Fprintf(os.Stderr, "Receiver user cannot be found!\n")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Если ошибка не в отсутствии пользователя - падаем и выходим
		fmt.Fprintf(os.Stderr, "Get receiver user balance QueryRow failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("Receiver user with ID %d pre-transfer balance is %d", TransferRequestUnmarshalRes.ReceiverUserID, currentReceiverAmount)

	// Выходим, если баланс отправителя не позволяет отправить желаемую сумму
	if TransferRequestUnmarshalRes.Amount > currentSenderAmount {
		fmt.Fprintf(os.Stderr, "Sender balance is too low for this operation!\n")
		http.Error(w, "Sender balance is too low for this operation!", http.StatusForbidden)
		return
	} else {
		// Выставляем новый баланс пользователя-отправителя
		senderNewAmount := currentSenderAmount - TransferRequestUnmarshalRes.Amount

		_, err = c.Connection.Exec(context.Background(), "UPDATE users SET balance=$1 WHERE userid=$2", senderNewAmount, TransferRequestUnmarshalRes.SenderUserID)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Setting new sender balance failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		log.Printf("Sender user with ID %d new balance is %d", TransferRequestUnmarshalRes.SenderUserID, senderNewAmount)

		// Выставляем новый баланс пользователя-получателя
		receiverNewAmount := currentReceiverAmount + TransferRequestUnmarshalRes.Amount
		_, err = c.Connection.Exec(context.Background(), "UPDATE users SET balance=$1 WHERE userid=$2", receiverNewAmount, TransferRequestUnmarshalRes.ReceiverUserID)

		log.Printf("Receiver user with ID %d new balance is %d", TransferRequestUnmarshalRes.SenderUserID, receiverNewAmount)

		if err != nil {
			fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// Формируем ответ и возвращаем его
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		balanceAnswer := models.SingleUserRequest{UserID: TransferRequestUnmarshalRes.ReceiverUserID, UserBalance: receiverNewAmount}

		jsonResp, err := json.Marshal(balanceAnswer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error happened in JSON marshal. Err: %s\n", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Write(jsonResp)

	}
}

// Функция, которая возвращает баланс пользователя
func (c *DBConnection) BalanceHandler(w http.ResponseWriter, r *http.Request) {
	balanceActionId, err := strconv.Atoi(r.URL.Query().Get("id"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error happened while searching id parameter. Err: %s\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем текущий баланс пользователя и записываем его в переменную
	var currentUserAmount int
	err = c.Connection.QueryRow(context.Background(), "SELECT balance FROM users WHERE userid=$1", balanceActionId).Scan(&currentUserAmount)

	// Если пользователь не найден - выходим
	if errors.Is(err, pgx.ErrNoRows) {
		fmt.Fprintf(os.Stderr, "Cannot get balance - user %d cannot be found in DB: %v\n", balanceActionId, err)
		http.Error(w, "Cannot get balance - user "+strconv.Itoa(balanceActionId)+" cannot be found in DB!", http.StatusNotFound)
		return
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		// Если ошибка не в отсутствии пользователя - падаем и выходим
		fmt.Fprintf(os.Stderr, "Get user balance QueryRow failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("Requested user with ID %d has balance %d", balanceActionId, currentUserAmount)

	// Формируем ответ и возвращаем его
	balanceAnswer := models.SingleUserRequest{UserID: balanceActionId, UserBalance: currentUserAmount}

	jsonResp, err := json.Marshal(balanceAnswer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error happened in JSON marshal for BalanceHandler answer. Err: %s\n", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}
