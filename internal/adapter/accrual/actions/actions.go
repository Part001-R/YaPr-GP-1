package actionsaccr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"go.uber.org/zap"
)

func (a *AccrualT) GetOrderInfo(order string) (OrderDataRxT, error) {

	// Проверка аргументов
	if order == "" {
		return OrderDataRxT{}, errors.New("в аргументе order нет содержимого")
	}

	// Подготовка
	url := fmt.Sprintf(a.Address+"/api/orders/"+"%s", order)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return OrderDataRxT{}, fmt.Errorf("ошибка при подготовке запроса: <%w>", err)
	}

	// Запрос
	resp, err := client.Do(req)
	if err != nil {
		return OrderDataRxT{}, fmt.Errorf("ошибка запроса: <%w>", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Log.Error("Ошибка закрытия тела ответа",
				zap.String("code", err.Error()),
			)
		}
	}()

	//---------------

	// Обработка ответа
	switch resp.StatusCode {
	case http.StatusOK: // 200 - успешная обработка запроса

		bodyRx, err := io.ReadAll(resp.Body)
		if err != nil {
			return OrderDataRxT{}, fmt.Errorf("ошибка при чтении тела ответа: <%w>", err)
		}

		var orderResponse OrderDataRxT
		if err := json.Unmarshal(bodyRx, &orderResponse); err != nil {
			return OrderDataRxT{}, fmt.Errorf("ошибка Unmarshal: <%w>", err)
		}
		orderResponse.Order = strings.Trim(orderResponse.Order, "<>") // по ТЗ, в ответе -> "order": "<number>",
		return orderResponse, nil

	case http.StatusNoContent: // 204 - заказ не зарегистрирован в системе расчёта
		return OrderDataRxT{}, fmt.Errorf("%s", ErrNotExistOrder)

	case http.StatusTooManyRequests: // 429 - превышено количество запросов к сервису
		return OrderDataRxT{}, fmt.Errorf("%s", ErrTooManyRequests)

	case http.StatusInternalServerError: // 500 - внутренняя ошибка сервера
		return OrderDataRxT{}, fmt.Errorf("%s", ErrInternalServerErr)

	default:
		logger.Log.Error("Неизвестный код ответа",
			zap.String("code", fmt.Sprintf("%d", resp.StatusCode)),
		)
		return OrderDataRxT{}, fmt.Errorf("%d", resp.StatusCode)
	}
}
