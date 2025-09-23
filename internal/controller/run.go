package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Функция запуска контроллера.
//
// Параметры:
//
// params - параметры.
// chControllerErr - канал для возврата ошибки работы.
func RunController(params *ControllerT, chControllerErr chan error) {

	cr := chi.NewRouter()

	// Точки входа
	routers(cr, params)

	// Запуск сервера
	srvConf := &http.Server{
		Addr:    params.Flags.RunAddress,
		Handler: cr,
	}

	if err := startUpHTTPServer(srvConf); err != nil {
		chControllerErr <- err
	}
}

// В функции содержатся точки входа для HTTP запросов.
//
// Параметры:
//
// cr - роутер.
// params - параметры.
func routers(cr *chi.Mux, params *ControllerT) {

	// Регистрация пользователя
	cr.Post("/api/user/register", Middleware(http.HandlerFunc(params.Register)))

	// Аутентификация пользователя
	cr.Post("/api/user/login", Middleware(http.HandlerFunc(params.Login)))

	// Добавление заказа
	cr.Post("/api/user/orders", Middleware(http.HandlerFunc(params.AddOrder)))

	// Списание средств
	cr.Post("/api/user/balance/withdraw", Middleware(http.HandlerFunc(params.BalanceWithdraw)))

	// Получение списка загруженных номеров заказов
	cr.Get("/api/user/orders", Middleware(http.HandlerFunc(params.GetOrdersUser)))

	// Получение текущего баланса пользователя
	cr.Get("/api/user/balance", Middleware(http.HandlerFunc(params.GetUserBalance)))

	// Получение информации о выводе средств
	cr.Get("/api/user/withdrawals", Middleware(http.HandlerFunc(params.GetHistoryWithdrawals)))

	// ---

	// Обработчик по умолчанию
	cr.NotFound(http.HandlerFunc(DefaultHandler))
}

// Функция выполняет запуск HTTP сервера.
//
// Парметры:
//
// srv - настройки сервера.
// txErr - канал для возврата ошибки.
func startUpHTTPServer(srv *http.Server) error {

	// Проверка параметров
	if srv == nil {
		return errors.New("в параметре srv, нет указателя")
	}

	// Запуск
	logger.Log.Info("Запуск сервера", zap.String("address", srv.Addr))

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Log.Error("Ошибка при запуске сервера", zap.Error(err))
	}
	return fmt.Errorf("ошибка при запуске сервера: <%w>", err)
}
