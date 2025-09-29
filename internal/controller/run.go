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
func RunController(params *ControllerConf, chControllerErr chan error) {

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
func routers(cr *chi.Mux, params *ControllerConf) {

	// Для аутентифицированных пользователей
	cr.Group(func(r chi.Router) {
		r.Use(authorizationMiddleware)
		r.Use(encodingMiddleware)

		// Добавление заказа
		r.Post("/api/user/orders", http.HandlerFunc(params.AddOrder))

		// Списание средств
		r.Post("/api/user/balance/withdraw", http.HandlerFunc(params.BalanceWithdraw))

		// Получение списка загруженных номеров заказов
		r.Get("/api/user/orders", http.HandlerFunc(params.GetOrdersUser))

		// Получение текущего баланса пользователя
		r.Get("/api/user/balance", http.HandlerFunc(params.GetUserBalance))

		// Получение информации о выводе средств
		r.Get("/api/user/withdrawals", http.HandlerFunc(params.GetHistoryWithdrawals))
	})

	// Регистрация пользователя
	cr.Post("/api/user/register", http.HandlerFunc(encodingMiddleware(http.HandlerFunc(params.Register)).ServeHTTP))

	// Аутентификация пользователя
	cr.Post("/api/user/login", http.HandlerFunc(encodingMiddleware(http.HandlerFunc(params.Login)).ServeHTTP))
}

// Функция выполняет запуск HTTP сервера. Возвращается ошибка.
//
// Парметры:
//
// srv - настройки сервера.
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
