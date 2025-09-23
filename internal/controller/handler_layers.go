package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Part001-R/YaPr-GP-1/internal/service/actions"
	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"go.uber.org/zap"
)

// UserRegister (POST)

func UserRegisterLayerRx(r *http.Request) (RegisterRxT, error) {

	// Проверка аргументов
	if r == nil {
		logger.Log.Error("в аргументе r нет указателя")
		return RegisterRxT{}, fmt.Errorf("%d", http.StatusInternalServerError)
	}
	if r.Header.Get("Content-Type") != "application/json" {
		return RegisterRxT{}, fmt.Errorf("%d", http.StatusBadRequest)
	}

	// Чтение тела запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Ошибка при чтении тела запроса",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		return RegisterRxT{}, fmt.Errorf("%d", http.StatusInternalServerError)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			logger.Log.Error("Ошибка при закрытии r.Body",
				zap.Error(err),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
		}
	}()

	var rxData RegisterRxT
	if err := json.Unmarshal(bodyBytes, &rxData); err != nil {
		logger.Log.Error("Ошибка Unmarshal",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		return RegisterRxT{}, fmt.Errorf("%d", http.StatusBadRequest)
	}

	// Проверка на пустое значение
	if rxData.Login == "" || rxData.Password == "" {
		return RegisterRxT{}, fmt.Errorf("%d", http.StatusBadRequest)
	}

	// Результат
	return rxData, nil
}

func UserRegisterLayerTx(w http.ResponseWriter) error {

	// Проверка аргументов
	if w == nil {
		return errors.New("в аргументе w нет указателя")
	}

	// Результат
	w.WriteHeader(http.StatusOK)

	return nil
}

// UserLogin (POST)

func UserLoginLayerRx(r *http.Request) (LoginRxT, error) {

	// Проверка аргументов
	if r == nil {
		logger.Log.Error("в аргументе r нет указателя")
		return LoginRxT{}, fmt.Errorf("%d", http.StatusInternalServerError)
	}
	if r.Header.Get("Content-Type") != "application/json" {
		return LoginRxT{}, fmt.Errorf("%d", http.StatusBadRequest)
	}

	// Чтение тела запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Ошибка при чтении тела запроса",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		return LoginRxT{}, fmt.Errorf("%d", http.StatusInternalServerError)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			logger.Log.Error("Ошибка при закрытии r.Body",
				zap.Error(err),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
		}
	}()

	var rxData LoginRxT
	if err := json.Unmarshal(bodyBytes, &rxData); err != nil {
		logger.Log.Error("Ошибка Unmarshal",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		return LoginRxT{}, fmt.Errorf("%d", http.StatusBadRequest)
	}

	// Проверка на пустое значение
	if rxData.Login == "" || rxData.Password == "" {
		return LoginRxT{}, fmt.Errorf("%d", http.StatusBadRequest)
	}

	// Результат
	return rxData, nil
}

func UserLoginLayerTx(w http.ResponseWriter, token string) error {

	// Проверка аргументов
	if w == nil {
		return errors.New("в аргументе w нет указателя")
	}
	if token == "" {
		return errors.New("в аргументе token нет содержимого")
	}

	// Возврат токена
	w.Header().Set("Authorization", token)
	w.WriteHeader(http.StatusOK)

	return nil
}

// UserOrders (POST)

func AddOrderLayerRx(r *http.Request) (order, token string, err error) {

	// Проверка аргументов
	if r == nil {
		logger.Log.Error("в аргементе r нет указателя")
		return "", "", fmt.Errorf("%d", http.StatusInternalServerError)
	}
	if r.Header.Get("Content-Type") != "text/plain" {
		return "", "", fmt.Errorf("%d", http.StatusBadRequest)
	}

	// Чтение тела запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Ошибка при чтении тела запроса",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		return "", "", fmt.Errorf("%d", http.StatusInternalServerError)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			logger.Log.Error("Ошибка при закрытии r.Body",
				zap.Error(err),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
		}
	}()

	order = string(bodyBytes)

	// Проверка на пустое значение
	order = strings.Trim(order, "\"")
	if order == "" {
		return "", "", fmt.Errorf("%d", http.StatusBadRequest)
	}

	token = r.Header.Get("Authorization")
	if token == "" {
		return "", "", fmt.Errorf("%d", http.StatusUnauthorized)
	}

	// Результат
	return order, token, nil
}

func AddOrderLayerTx(w http.ResponseWriter) error {

	// Проверка аргументов
	if w == nil {
		return errors.New("в аргументе w нет указателя")
	}

	w.WriteHeader(http.StatusAccepted)

	return nil
}

// UserOrders (GET)

func GetOrdersUserLayerRx(r *http.Request) (token string, err error) {

	// Проверка аргументов
	if r == nil {
		logger.Log.Error("в аргументе r нет указателя")
		return "", fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Логика
	token = r.Header.Get("Authorization")

	if token == "" {
		return "", fmt.Errorf("%d", http.StatusUnauthorized)
	}

	// Результат
	return token, nil
}

func GetOrdersUserLayerTx(w http.ResponseWriter, orders []actions.OrderT) error {

	// Проверка аргументов
	if w == nil {
		logger.Log.Error("в аргементе w нет указателя")
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}
	if orders == nil {
		logger.Log.Error("в аргементе orders нет указателя")
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Логика
	byteTx, err := json.Marshal(orders)
	if err != nil {
		logger.Log.Error("Ошибка Marshal",
			zap.String("err", err.Error()),
		)
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(byteTx)

	return nil
}

// userBalance (GET)

func GetUserBalanceLayerRx(r *http.Request) (token string, err error) {

	// Проверка аргументов
	if r == nil {
		logger.Log.Error("в аргументе r нет указателя")
		return "", fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Логика
	token = r.Header.Get("Authorization")

	if token == "" {
		return "", fmt.Errorf("%d", http.StatusUnauthorized)
	}

	// Результат
	return token, nil
}

func GetUserBalanceLayerTx(w http.ResponseWriter, balanceRx actions.BalanceT) error {

	// Проверка аргументов
	if w == nil {
		logger.Log.Error("в аргементе w нет указателя")
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Логика
	byteTx, err := json.Marshal(balanceRx)
	if err != nil {
		logger.Log.Error("Ошибка Marshal",
			zap.String("err", err.Error()),
		)
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(byteTx)

	return nil
}

// BalanceWithdraw (POST)

func BalanceWithdrawLayerRx(r *http.Request) (string, actions.BalanceWithdrawT, error) {

	// Проверка аргументов
	if r == nil {
		logger.Log.Error("в аргументе r нет указателя")
		return "", actions.BalanceWithdrawT{}, fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Логика
	token := r.Header.Get("Authorization")

	if token == "" {
		return "", actions.BalanceWithdrawT{}, fmt.Errorf("%d", http.StatusUnauthorized)
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Ошибка при чтении тела запроса",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		return "", actions.BalanceWithdrawT{}, fmt.Errorf("%d", http.StatusInternalServerError)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			logger.Log.Error("Ошибка при закрытии r.Body",
				zap.Error(err),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
		}
	}()

	var dataRx actions.BalanceWithdrawT

	if err := json.Unmarshal(bodyBytes, &dataRx); err != nil {
		logger.Log.Error("Ошибка Unmarshal",
			zap.Error(err),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		return "", actions.BalanceWithdrawT{}, fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Результат
	return token, dataRx, nil

}

func BalanceWithdrawLayerTx(w http.ResponseWriter) error {

	// Проверка аргументов
	if w == nil {
		return errors.New("в аргументе w нет указателя")
	}

	// Логика
	w.WriteHeader(http.StatusOK)

	return nil
}

// GetHistoryWithdrawels

func GetHistoryWithdrawelsLayerRx(r *http.Request) (token string, err error) {

	// Проверка аргументов
	if r == nil {
		logger.Log.Error("в аргументе r нет указателя")
		return "", fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Логика
	token = r.Header.Get("Authorization")

	if token == "" {
		return "", fmt.Errorf("%d", http.StatusUnauthorized)
	}

	// Результат
	return token, nil
}

func GetHistoryWithdrawelsLayerTx(w http.ResponseWriter, data []WithdrawalResponse) error {

	// Проверка аргументов
	if w == nil {
		logger.Log.Error("в аргементе w нет указателя")
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}
	if data == nil {
		logger.Log.Error("в аргементе data нет указателя")
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}
	if len(data) == 0 {
		logger.Log.Error("в аргументе data нет данных для отправки")
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Логика
	byteTx, err := json.Marshal(data)
	if err != nil {
		logger.Log.Error("Ошибка Marshal")
		return fmt.Errorf("%d", http.StatusInternalServerError)
	}

	// Ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(byteTx)

	return nil
}
