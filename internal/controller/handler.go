package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Part001-R/YaPr-GP-1/internal/service/actions"
	gz "github.com/Part001-R/YaPr-GP-1/internal/utils/gzip"

	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"go.uber.org/zap"
)

func authorizationMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			logger.Log.Info("принят неавторизованный запрос",
				zap.String("URI", r.RequestURI),
				zap.String("метод", r.Method),
			)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func encodingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		// Проверка поддерживает ли сервер запрашиваемую клиентом кодировку
		acceptEncoding := r.Header.Get("Accept-Encoding")
		found := false

		if acceptEncoding != "" {
			encodings := strings.Split(acceptEncoding, ",")

			for _, v := range encodings {
				encodingType := strings.TrimSpace(v)

				switch encodingType {
				case "gzip":
					cw := gz.NewCompressWriter(w)
					ow = cw
					defer func() {
						if err := cw.Close(); err != nil {
							logger.Log.Error("Ошибка при закрытии cw", zap.Error(err))
						}
					}()
					found = true
				case "identity":
					found = true
				default:
				}
			}

			if !found {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}

		// Проверка, как клиент закодировал переданные данные
		contentEncoding := r.Header.Get("Content-Encoding")
		found = false

		if contentEncoding != "" {
			encodings := strings.Split(contentEncoding, ",")

			for _, v := range encodings {
				encodingType := strings.TrimSpace(v)

				switch encodingType {
				case "gzip":
					cr, err := gz.NewCompressReader(r.Body)
					if err != nil {
						logger.Log.Error("Ошибка в NewCompressReader",
							zap.Error(err),
							zap.String("method", r.Method),
							zap.String("url", r.URL.String()),
						)
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
					defer func() {
						if err := cr.Close(); err != nil {
							logger.Log.Error("Ошибка при закрытии cr", zap.Error(err))
						}
					}()
					defer func() {
						if err := r.Body.Close(); err != nil {
							logger.Log.Error("Ошибка при закрытии r.Body", zap.Error(err))
						}
					}()

					r.Body = cr
					found = true
				case "identity":
					found = true
				default:
				}
			}

			if !found {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}

		// Запуск обработчика
		timeStart := time.Now()
		h.ServeHTTP(ow, r) // Используем ServeHTTP для передачи управления
		duration := time.Since(timeStart)

		// Вывод в лог сводной информации по запросу
		logger.Log.Info("принят HTTP запрос",
			zap.String("URI", r.RequestURI),
			zap.String("метод", r.Method),
			zap.Duration("время выполнения запроса", duration),
		)
	})
}

// Регистрация пользователя
func (c *ControllerConf) Register(w http.ResponseWriter, r *http.Request) {

	// Приём
	registerDataRx, err := UserRegisterLayerRx(r)
	if err != nil {
		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case "400":
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		default:
			logger.Log.Error("Код ошибки неопознан",
				zap.String("code", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Работа
	token, err := c.ServAct.RegistrationUser(registerDataRx.Login, registerDataRx.Password)
	if err != nil {

		logger.Log.Error("Функция сервиса RegistrationUser, вернула ошибку",
			zap.String("err", err.Error()),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)

		baseErr := unwrapErr(err)
		if baseErr.Error() == errUserExist { // "pq: duplicate key value violates unique constraint \"users_user_name_key\""
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Ответ
	if err := UserRegisterLayerTx(w, token); err != nil {
		logger.Log.Error("Ошибка при формировании ответа",
			zap.String("err", err.Error()),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// Аутентификация пользователя
func (c *ControllerConf) Login(w http.ResponseWriter, r *http.Request) {

	// Приём
	loginDataRx, err := UserLoginLayerRx(r)
	if err != nil {
		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case "400":
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		default:
			logger.Log.Error("Код ошибки неопознан",
				zap.String("code", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Логика
	token, err := c.ServAct.AuthenticationUser(loginDataRx.Login, loginDataRx.Password)
	if err != nil {
		logger.Log.Error("Функция сервиса AuthenticationUser, вернула ошибку",
			zap.String("err", err.Error()),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)

		baseErr := unwrapErr(err)
		if baseErr.Error() == errPairLoginPassword { // по ТЗ
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		if baseErr.Error() == errUserNotFound {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Возврат
	if err := UserLoginLayerTx(w, token); err != nil {
		logger.Log.Error("Ошибка при формировании ответа",
			zap.String("err", err.Error()),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// Загрузка номера заказа
func (c *ControllerConf) AddOrder(w http.ResponseWriter, r *http.Request) {

	// Приём
	order, tokenRx, err := AddOrderLayerRx(r)
	if err != nil {

		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case "400":
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		case "401":
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		default:
			logger.Log.Error("Код ошибки неопознан",
				zap.String("code", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Логика
	if err := c.ServAct.AddOrder(tokenRx, order); err != nil {

		baseErr := unwrapErr(err)

		switch baseErr.Error() {
		case errOrderExist: // номер заказа уже был загружен этим пользователем
			http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
			return
		case errNoAuthorization: // пользователь не авторизован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		case errNoAuthentication: // пользователь не аутентифицирован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		case errOrderBusy: // номер заказа уже был загружен другим пользователем
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		case errOrderFormat: // неверный формат номера заказа
			http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			return
		default:

			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Возврат
	if err := AddOrderLayerTx(w); err != nil {
		logger.Log.Error("Ошибка при формировании ответа",
			zap.String("err", err.Error()),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// Получение списка загруженных номеров заказов
func (c *ControllerConf) GetOrdersUser(w http.ResponseWriter, r *http.Request) {

	// Приём
	tokenRx, err := GetOrdersUserLayerRx(r)
	if err != nil {
		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case "401":
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		default:
			logger.Log.Error("Код ошибки неопознан",
				zap.String("code", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Логика
	orders, err := c.ServAct.GetOrdersUser(tokenRx)
	if err != nil {

		baseErr := unwrapErr(err)

		switch baseErr.Error() {
		case errOrdersNoContent: // нет данных для ответа
			http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
			return
		case errNoAuthorization: // пользователь не авторизован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		case errNoAuthentication: // пользователь не аутентифицирован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		default:
			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Ответ
	if err := GetOrdersUserLayerTx(w, orders); err != nil {

		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		default:
			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

// Получение текущего баланса пользователя
func (c *ControllerConf) GetUserBalance(w http.ResponseWriter, r *http.Request) {

	// Приём
	token, err := GetUserBalanceLayerRx(r)
	if err != nil {
		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case "401":
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		default:
			logger.Log.Error("Код ошибки неопознан",
				zap.String("code", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Логика
	balanceRx, err := c.ServAct.GetUserBalance(token)
	if err != nil {

		baseErr := unwrapErr(err)

		switch baseErr.Error() {
		case errNoAuthorization: // пользователь не авторизован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		case errNoAuthentication: // пользователь не аутентифицирован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		case errNotFoundBalance: // данные баланса пользователя не найдены
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		default:
			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Возврат
	if err := GetUserBalanceLayerTx(w, balanceRx); err != nil {

		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		default:
			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

// Cписание средств
func (c *ControllerConf) BalanceWithdraw(w http.ResponseWriter, r *http.Request) {

	// Приём
	token, dataRx, err := BalanceWithdrawLayerRx(r)
	if err != nil {
		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		default:
			logger.Log.Error("Код ошибки неопознан",
				zap.String("code", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Логика
	var copyDataRx actions.BalanceWithdraw
	copyDataRx.Order = dataRx.Order
	copyDataRx.Sum = dataRx.Sum

	if err := c.ServAct.BalanceWithdraw(token, copyDataRx); err != nil {

		baseErr := unwrapErr(err)

		switch baseErr.Error() {
		case errNoAuthorization: // пользователь не авторизован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		case errNoAuthentication: // пользователь не аутентифицирован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		case errNotEnoughtBalance: //  на счету недостаточно средств
			http.Error(w, http.StatusText(http.StatusPaymentRequired), http.StatusPaymentRequired)
			return
		case errIncorrectOrderNumb, errWithdrOrderExist: //  неверный номер заказа
			http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			return
		default:
			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Ответ
	if err := BalanceWithdrawLayerTx(w); err != nil {
		logger.Log.Error("Ошибка при формировании ответа",
			zap.String("err", err.Error()),
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
		)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// Получение истории списания
func (c *ControllerConf) GetHistoryWithdrawals(w http.ResponseWriter, r *http.Request) {

	// Приём
	token, err := GetHistoryWithdrawelsLayerRx(r)
	if err != nil {
		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case "401":
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		default:
			logger.Log.Error("Код ошибки неопознан",
				zap.String("code", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Логика
	history, err := c.ServAct.HistoryWithdrawels(token)
	if err != nil {

		baseErr := unwrapErr(err)

		switch baseErr.Error() {
		case errNoWithdrawels: // нет ни одного списания
			http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
			return
		case errNoAuthorization: // пользователь не авторизован
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		default:
			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	// Подготовка данных для ответа
	dataTx, err := prepareWithdrawalResponse(history)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Ответ
	if err := GetHistoryWithdrawelsLayerTx(w, dataTx); err != nil {

		switch err.Error() {
		case "500":
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		default:
			logger.Log.Error("Ошибка неопознана",
				zap.String("err", err.Error()),
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

// Разворачивание всех оборачиваний. Возвращается исходная ошибка.
//
// Параметры:
//
// err - принятое сообщение ошибки.
func unwrapErr(err error) error {

	originalErr := err
	for {
		unwrappedErr := errors.Unwrap(originalErr)
		if unwrappedErr == nil {
			break
		}
		originalErr = unwrappedErr
	}
	return originalErr
}

// Функция подготавливает данные для отправки. Возвращает подготовленные данные и ошибку.
//
// Парамметры:
//
// withdrawals - история списаний.
func prepareWithdrawalResponse(withdrawals []actions.HistoryWithdrawals) ([]WithdrawalResponse, error) {

	// Проверка аргументов
	if withdrawals == nil {
		return nil, errors.New("нет указателя в аргументе withdrawals")
	}

	// Логика
	var response []WithdrawalResponse

	for _, withdrawal := range withdrawals {
		response = append(response, WithdrawalResponse{
			Order:       withdrawal.Order,
			Sum:         withdrawal.Sum,
			ProcessedAt: withdrawal.ProcessedAt.Format(time.RFC3339),
		})
	}

	return response, nil
}
