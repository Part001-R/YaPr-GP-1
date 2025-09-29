package actionsaccr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetOrderInfo

func Test_GetOrderInfo_SUCCESS(t *testing.T) {

	// Сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		dataTx := OrderDataRx{
			Order:   "<12345>",
			Status:  "PROCESSED",
			Accrual: 500,
		}

		byteTx, err := json.Marshal(dataTx)
		require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

		w.WriteHeader(http.StatusOK)
		w.Write(byteTx)
	}))
	defer ts.Close()

	// Экземпляр адаптера
	accrI := NewInstAdapterAccrual(ts.URL)

	// Данные для теста
	tests := []struct {
		name      string
		order     string
		wantData  OrderDataRx
		wantError error
	}{
		{
			name:  "Successful response",
			order: "12345",
			wantData: OrderDataRx{
				Order:   "12345",
				Status:  "PROCESSED",
				Accrual: 500,
			},
			wantError: nil,
		},
	}

	// Тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result, err := accrI.GetOrderInfo(tt.order)
			require.NoErrorf(t, err, "неожиданная ошибка запроса: <%v>", err)

			result.Order = strings.Trim(result.Order, "<>")

			assert.Equalf(t, tt.wantData.Order, result.Order, "ожидался Order <%s> а принято <%s>", tt.wantData.Order, result.Order)
			assert.Equalf(t, tt.wantData.Status, result.Status, "ожидался Status <%s> а принято <%s>", tt.wantData.Status, result.Status)
			assert.Equalf(t, tt.wantData.Accrual, result.Accrual, "ожидался Accrual <%s> а принято <%s>", tt.wantData.Accrual, result.Accrual)
		})
	}
}

func Test_GetOrderInfo_FAULT(t *testing.T) {

	// Создаем тестовый сервер
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch r.URL.Path {
		case "/api/orders/12345": // Успешный ответ

			dataTx := OrderDataRx{
				Order:   "<12345>",
				Status:  "PROCESSED",
				Accrual: 500,
			}
			byteTx, err := json.Marshal(dataTx)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			w.WriteHeader(http.StatusOK)
			w.Write(byteTx)

		case "/api/orders/67890": // Заказ не найден
			w.WriteHeader(http.StatusNoContent)

		case "/api/orders/429": // Превышено количество запросов
			w.WriteHeader(http.StatusTooManyRequests)

		case "/api/orders/500": // Внутренняя ошибка сервера
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	// Экземпляр адаптера
	accrI := NewInstAdapterAccrual(ts.URL)

	// Данные для теста
	tests := []struct {
		name      string
		order     string
		wantError error
	}{
		{
			name:      "заказ не найден",
			order:     "67890",
			wantError: errors.New("заказ не зарегистрирован в системе расчёта"),
		},
		{
			name:      "Много запросов",
			order:     "429",
			wantError: errors.New("превышено количество запросов к сервису"),
		},
		{
			name:      "Ошибка сервера",
			order:     "500",
			wantError: errors.New("внутренняя ошибка сервера"),
		},
		{
			name:      "Нет номера заказа",
			order:     "",
			wantError: errors.New("в аргументе order нет содержимого"),
		},
	}

	// Тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := accrI.GetOrderInfo(tt.order)
			assert.EqualError(t, err, tt.wantError.Error())
		})
	}
}
