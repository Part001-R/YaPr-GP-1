package service

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	actionsaccr "github.com/Part001-R/YaPr-GP-1/internal/adapter/accrual/actions"
	actionspg "github.com/Part001-R/YaPr-GP-1/internal/adapter/postgres/actions"
	"github.com/Part001-R/YaPr-GP-1/internal/controller"
	"github.com/Part001-R/YaPr-GP-1/internal/service/actions"
	"github.com/Part001-R/YaPr-GP-1/internal/utils/database"
	"github.com/Part001-R/YaPr-GP-1/internal/utils/flags"
	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"go.uber.org/zap"
)

const (
	logLevel = "info"
)

// Запуск сервиса
func App() error {

	// Подготовительные действия
	params, adptAccr, adptPG, err := prepare()
	if err != nil {
		return fmt.Errorf("ошибка в функции prepare: <%w>", err)
	}

	// Запуск
	err = server(params, adptPG, adptAccr)
	if err != nil {
		return fmt.Errorf("ошибка в функции server: <%w>", err)
	}

	return nil
}

// Подготовительные действия
func prepare() (*ServiceConf, actionsaccr.Accrual, actionspg.Postgres, error) {

	// Флаги
	flags := flags.ParseFlags()

	// Логер
	err := logger.Initialize(logLevel)
	if err != nil {
		return &ServiceConf{}, nil, nil, fmt.Errorf("функия Initialize вернула ошибку: <%w>", err)
	}

	// БД
	if flags.DatabaseURI == "" {
		return &ServiceConf{}, nil, nil, errors.New("нет содержимого в DatabaseURI")
	}

	dbPtr, funcCloseDB, err := database.ConnectDB(flags.DatabaseURI)
	if err != nil {
		return &ServiceConf{}, nil, nil, fmt.Errorf("функция ConnectDB вернула ошибку: <%w>", err)
	}

	if err := database.MigrationUpDB(dbPtr); err != nil {
		return &ServiceConf{}, nil, nil, fmt.Errorf("функия MigrationDB вернула ошибку: <%w>", err)
	}

	// Экземляр адаптера Postgres
	adptPg := actionspg.NewInstAdapterPostgres(dbPtr)

	// Экземляр адаптера Accrual
	adptAccr := actionsaccr.NewInstAdapterAccrual(flags.AccuralSystemAddress)

	// Экземпляр интерфейса сервиса
	instAct := actions.NewInstServiceActionsT(adptPg, adptAccr)
	servAct := actions.NewInstServiceActionsI(instAct)

	// Экземпляр контроллера
	paramsCntrl := controller.NewInstController(flags, servAct)

	// Экземпляр сервиса
	paramsService := NewInstService(dbPtr, instAct.ChAccrNewOrder, funcCloseDB, paramsCntrl)

	return paramsService, adptAccr, adptPg, nil
}

// Работа
func server(params *ServiceConf, adptPG actionspg.Postgres, adptAccr actionsaccr.Accrual) error {

	// Проверка аргументов
	if params == nil {
		return errors.New("в параметре params, нет указателя")
	}

	// Проврка значений id в таблицах БД
	if err := checkIDTables(adptPG); err != nil {
		return fmt.Errorf("функция checkIDTables вернула ошибку: <%w>", err)
	}

	// Проверка номеров заказа на статус NEW и перенос в очередь
	if err := startUpCheckOrdersByNEW(adptPG); err != nil {
		return fmt.Errorf("функция startUpCheckOrdersByNEW вернула ошибку: <%w>", err)
	}

	// Запуск контроллера
	chCntrErr := make(chan error)
	go controller.RunController(params.Cntr, chCntrErr)

	// Запуск обработчика очереди запросов к Accrual
	chAccrErr := make(chan error)
	go runAccrual(*params.ChAccr.NewOrderNumb, adptAccr, adptPG, chAccrErr)

	// Сигналы остановки
	sigSys := make(chan os.Signal, 1)
	signal.Notify(sigSys, syscall.SIGINT, syscall.SIGTERM)

	data := checkReasonStop{
		chCntrErr: chCntrErr,
		chAccrErr: chAccrErr,
		sigSys:    sigSys,
	}

	// Отслеживание причины остановки
	if err := signalsStopRun(data, params); err != nil {
		logger.Log.Error("функция signalsStopRun вернула ошибку", zap.String("ошибка", err.Error()))
		return fmt.Errorf("функция signalsStopRun вернула ошибку: <%w>", err)
	}

	return nil
}

// Функция определяет причину остановки выполнения. Возвращается ошибка.
//
// Параметры:
//
// data - набор данных для обеспечения работы функции.
// params - параметры.
func signalsStopRun(data checkReasonStop, params *ServiceConf) error {

	// Проверка аргументов
	if data.chAccrErr == nil {
		logger.Log.Error("в аргументе data.chAccrErr нет указателя")
		return errors.New("в аргументе data.chAccrErr нет указателя")
	}
	if data.chCntrErr == nil {
		logger.Log.Error("в аргументе data.chCntrErr нет указателя")
		return errors.New("в аргументе data.chCntrErr нет указателя")
	}
	if data.sigSys == nil {
		logger.Log.Error("в аргументе data.sigSys нет указателя")
		return errors.New("в аргументе data.sigSys нет указателя")
	}
	if params == nil {
		logger.Log.Error("в аргументе params нет указателя")
		return errors.New("в аргументе params нет указателя")
	}

	// Закрытие подключения к БД
	defer params.DB.FuncClose()

	// Проверка на nil для полей структуры
	if data.sigSys == nil {
		return errors.New("канал sigSys не инициализирован")
	}
	if data.chCntrErr == nil {
		return errors.New("канал chCntrErr не инициализирован")
	}

	// Логика
	select {
	case <-data.sigSys:
		logger.Log.Info("сервер остановлен штатно")
		return nil
	case err := <-data.chCntrErr:
		logger.Log.Error("ошибка контроллера", zap.String("ошибка", err.Error()))
		return err
	case err := <-data.chAccrErr:
		logger.Log.Error("ошибка Accrual", zap.String("ошибка", err.Error()))
		return err
	}
}

// Функция взаимодействия с системой начисления баллов лояльности. Запускается в go рутине.
//
// Параметры:
//
// chRx - канал для приёма нового номера заказа.
// adptAccr - интерфес адаптера Accrual.
// adptPG - интерфейс адаптера Postgres.
// chAccrErr - канал для возврата ошибки go рутины.
func runAccrual(chRx chan string, adptAccr actionsaccr.Accrual, adptPG actionspg.Postgres, chAccrErr chan error) {

	/*
		Логика работы.

		Реализовано взаимодействие с функцией обработчика запросов - processingOrder, через каналы.
		Запуск функции происходит при поступлении данных из канала HTTP или Queue.

		При запуске runAccrual, происходит запуск вспомогательной go рутины - runOrdersQueue.
		runOrdersQueue, периодически проверяет таблицу ожидающих на обработку номеров заказов.
		Если есть ожидающие номера, то последовательно начинается их передача в канал Queue, для обработки.

		В случае получения от Accrual статуса заказа "INVALID" или "PROCESSED", заказ считается обработанным.
		Происходит обновление данных заказа и удаления из очереди.

		При возникновении ошибки, ошибка передаётся в канал chAccrErr. Работа сервиса останавливается.
	*/

	// Проверка аргументов
	if chAccrErr == nil {
		log.Fatal("Работа прервана. В аргументе chAccrErr, нет указателя")
	}
	if chRx == nil {
		chAccrErr <- errors.New("в аргементе chRx, нет указателя")
	}
	if adptAccr == nil {
		chAccrErr <- errors.New("в аргементе adptAccr, нет указателя")
	}
	if adptPG == nil {
		chAccrErr <- errors.New("в аргементе adptPG, нет указателя")
	}

	// Запуск выборки каналов из очереди
	chRxQueue := make(chan string)
	go runOrdersQueue(chRxQueue, adptPG, chAccrErr)

	// Логика
	paramsProcOrdrQue := ProcessingOrder{
		tmr:                   &time.Timer{},
		newOrdNumb:            "",
		needStoreByFreq:       false,
		adptAccr:              adptAccr,
		adptPG:                adptPG,
		needStoreByErr:        false,
		needStoreByNotReg:     false,
		needStoreByREGISTERED: false,
		needStoreByPROCESSING: false,
	}

	for {
		select {
		// заказ из HTTP
		case paramsProcOrdrQue.newOrdNumb = <-chRx:
		// заказ из Queue
		case paramsProcOrdrQue.newOrdNumb = <-chRxQueue:
		}

		// Обработка заказа
		if err := processingOrder(&paramsProcOrdrQue); err != nil {
			chAccrErr <- fmt.Errorf("ошибка при обработке заказа: <%w>", err)
			return
		}
	}
}

// Функция выдаёт в канал номера заказов ожидающих обработки.
//
// Параметры:
//
// chTx - канал для передачи.
// adptPG - интерфейс адаптера Postgres.
// chAccrErr - канал для возврата ошибки.
func runOrdersQueue(chTx chan string, adptPG actionspg.Postgres, chAccrErr chan error) {

	// Проверка аргументов
	if chAccrErr == nil {
		log.Fatal("в аргументе chAccrErr нет указателя")
	}
	if chTx == nil {
		chAccrErr <- errors.New("в аргументе chTx, нет указателя")
		return
	}
	if adptPG == nil {
		chAccrErr <- errors.New("в аргументе adptPG, нет указателя")
		return
	}

	// Логика
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		// Получение списка заказов ожидающих обработки
		ordersQueue, err := adptPG.GetOrdersInQueue()
		if err != nil {
			chAccrErr <- fmt.Errorf("ошибка при получении списка ожидающих обработки заказов: <%w>", err)
			return
		}
		// Передача номеров заказа на обработку
		for _, v := range ordersQueue {
			select {
			case chTx <- v:
			case <-time.After(3 * time.Second):
				chAccrErr <- fmt.Errorf("отправленные данные в канал, не прочитаны за отведённое время")
				return
			}
		}
	}
}

// Функция проверяет ошибку на отсутствие подключение к Accrual. Возвращает true - если нет подключения.
//
// Параметры:
//
// err - исходная шибка
func isConnectionRefused(err error) bool {

	if err == nil {
		return false
	}

	errParts := strings.Split(err.Error(), ":")

	if len(errParts) > 0 {
		last := strings.TrimSpace(errParts[len(errParts)-1])
		last = strings.TrimSuffix(last, ">")

		if last == "connection refused" {
			return true
		}
	}
	return false
}

// Функция проверяет ошибку на timeout ответа от сервера. Возвращает true - если timeout ответа.
//
// Параметры:
//
// err - исходная шибка
func isResponceTimeout(err error) bool {

	if err == nil {
		return false
	}

	baseErr := errors.Unwrap(err)

	if baseErr == nil {
		return false
	}

	return strings.Contains(baseErr.Error(), ErrTimeoutResponceAccrual)
}

// Функция выполняет обработку заказа. Возвращает ошибку.
//
// Параметры:
//
// params - параметры.
func processingOrder(params *ProcessingOrder) (err error) {

	// Проверка аргументов
	if params == nil {
		return errors.New("в аргументе params, нет указателя")
	}

	// Логика
	select {
	case <-params.tmr.C: // Ожидание завершения времени выдержки по коду 429
		params.tmr.Stop()
		params.needStoreByFreq = false
	default:
	}

	var accrResponce actionsaccr.OrderDataRx

	// Если есть необходимость сохранения номера заказа в резервном хранилище, по коду 429
	if params.needStoreByFreq {
		if err := addOrderInQueue(params); err != nil {
			return fmt.Errorf("ошибка при сохранении заказа в резервной очереди: <%w>", err)
		}
		return nil
	}

	// Если нет необходимости сохранения номера заказа в резервном хранилище, по коду 429
	if !params.needStoreByFreq {

		// Запрос к системе Accrual
		accrResponce, err = params.adptAccr.GetOrderInfo(params.newOrdNumb)
		if err != nil {
			if er := checkErrors(err, params); er != nil {
				return fmt.Errorf("функция checkErrors, вернула ошибку: <%w>", er)
			}
			return nil
		}

		// Обработка ответа от Accrual
		if err := processingResponceAccrual(accrResponce, params); err != nil {
			return fmt.Errorf("функция processingResponseAccrual, вернула ошибку: <%w>", err)
		}
	}
	return nil
}

// Функция, вызывает функцию, с реализацией добавления заказа в очередь. Возвращает ошибку.
//
// Параметры:
//
// params - параметры.
func addOrderInQueue(params *ProcessingOrder) error {

	// Проверка аргументов
	if params == nil {
		return errors.New("в аргументе params, нет указателя")
	}

	if err := params.adptPG.AddOrderInQueue(params.newOrdNumb); err != nil {

		errBase := errors.Unwrap(err)
		if errBase.Error() == ErrOrderExistInQueue { // Если такой номер уже существует в очереди
			return nil
		}

		return fmt.Errorf("ошибка при сохранении заказа в резервной очереди: <%w>", err)
	}
	return nil
}

// Функция проверяет сетевые ошибки. Возвращает true - если сетевая ошибка и ошибку
//
// Параметры:
//
// er - обрабатываемая ошибка.
// params - параметры.
func checkNetErrors(err error) (bool, error) {

	// Проверка аргументов
	if err == nil {
		return false, fmt.Errorf("в аргументе err нет ошибки")
	}

	do := false

	// Отсутствие подключения к Accrual
	if isConnectionRefused(err) {
		logger.Log.Error("Нет подключения к Accrual. Заказ передан в очередь.")
		do = true
	}
	// Время ожидания ответа истекло
	if isResponceTimeout(err) {
		logger.Log.Error("Превышено время ответа от Accrual. Заказ передан в очередь.")
		do = true
	}

	return do, nil
}

// Функция проверяет оставшиеся возможные ошибки. Возвращает true - если есть распознование и ошибку.
//
// Параметры:
//
// err - обрабатываемая ошибка.
// params - параметры.
func checkOtherErrors(err error, params *ProcessingOrder) (bool, error) {

	// Проверка аргументов
	if err == nil {
		return false, fmt.Errorf("в аргументе err нет ошибки")
	}
	if params == nil {
		return false, fmt.Errorf("в аргументе params нет указателя")
	}

	// Логика
	do := false

	switch err.Error() {
	case ErrAccrNotExistOrder: // заказ не зарегистрирован в системе расчёта
		logger.Log.Error("Заказ не зарегистрирован в системе расчёта",
			zap.String("order", params.newOrdNumb),
		)
		params.needStoreByNotReg = true

	case ErrAccrTooManyRequests: // превышено количество запросов к сервису
		logger.Log.Error("Превышено количество запросов к сервису",
			zap.String("order", params.newOrdNumb),
		)
		if !params.needStoreByFreq { // защита инициализации
			params.needStoreByFreq = true
			params.tmr = time.NewTimer(10 * time.Second) // запуск таймера
		}

	case ErrAccrInternalServerErr: // Внутренняя ошибка сервера
		logger.Log.Error("Внутренняя ошибка сервера",
			zap.String("order", params.newOrdNumb),
		)
		params.needStoreByErr = true

	default:
		logger.Log.Error("Код ошибки неопознан",
			zap.String("err", err.Error()),
		)
		return false, fmt.Errorf("запрос к Accrual вернул неопознанную ошибку: <%w>", err)
	}

	// Проверка необходимости сохранения в очереди необработанных заказов
	// по кодам ответа
	if params.needStoreByErr || params.needStoreByFreq || params.needStoreByNotReg {

		params.needStoreByErr = false
		params.needStoreByNotReg = false

		do = true
	}

	return do, nil
}

// Функция выполняет проверку ошибки после выполнения запроса к Accrual. Возвращает ошибку.
//
// Параметры:
//
// err - обрабатываемая ошибка.
// params - параметры.
func checkErrors(err error, params *ProcessingOrder) error {

	// Проверка аргументов
	if err == nil {
		return fmt.Errorf("в аргументе err нет ошибки")
	}
	if params == nil {
		return fmt.Errorf("в аргументе params нет указателя")
	}

	// Логика
	//
	// Проверка сетевых ошибок
	flag, er := checkNetErrors(err)
	if er != nil {
		return fmt.Errorf("функция checkNetErrors, вернула ошибку: <%w>", er)
	}
	if flag {
		if err := addOrderInQueue(params); err != nil {
			return fmt.Errorf("ошибка при сохранении заказа в резервной очереди: <%w>", err)
		}
		return nil
	}

	// Проверка остальных ошибок
	flag, er = checkOtherErrors(err, params)
	if er != nil {
		return fmt.Errorf("функция checkOtherErrors, вернула ошибку: <%w>", er)
	}
	if flag {
		if err := addOrderInQueue(params); err != nil {
			return fmt.Errorf("ошибка при сохранении заказа в резервной очереди: <%w>", err)
		}
		return nil
	}

	return nil
}

// Функция выполняет обработку ответа от Accrual. Возвращает ошибку.
//
// Параметры:
//
// accrResponce - ответ от Accrual.
// params - параметры.
func processingResponceAccrual(accrResponce actionsaccr.OrderDataRx, params *ProcessingOrder) error {

	// Проверка статуса ответа
	switch accrResponce.Status {
	case "REGISTERED": // заказ зарегистрирован, но вознаграждение не рассчитано
		params.needStoreByREGISTERED = true
	case "PROCESSING": // расчёт начисления в процессе
		params.needStoreByPROCESSING = true
	default:
	}

	// Проверка необходимости сохранения в очереди необработанных заказов
	// по статусу ответа
	if params.needStoreByREGISTERED || params.needStoreByPROCESSING {

		params.needStoreByREGISTERED = false
		params.needStoreByPROCESSING = false

		// Добавление заказа в очередь необработанных
		if err := addOrderInQueue(params); err != nil {
			return fmt.Errorf("ошибка при сохранении заказа в резервной очереди: <%w>", err)
		}

		// Обновление статуса заказа
		if err := params.adptPG.UpdateOrderStatus(actionspg.DataOrderAccr(accrResponce)); err != nil {
			return fmt.Errorf("ошибка прb при обновлении статуса заказа: <%w>", err)
		}
		return nil
	}

	doUpdate := false

	switch accrResponce.Status {
	case "INVALID": // заказ не принят к расчёту, и вознаграждение не будет начислено
		doUpdate = true
	case "PROCESSED": // расчёт начисления окончен
		doUpdate = true
	default:
		logger.Log.Error("От Accrual принят неопознанный статус заказа",
			zap.String("status", accrResponce.Status),
		)
		if err := addOrderInQueue(params); err != nil {
			return fmt.Errorf("ошибка при сохранении заказа в резервной очереди: <%w>", err)
		}
		return nil
	}

	// Обновление данных заказа
	// Обновляется статус, баланс
	// Удаляется из очереди необработанных
	if doUpdate {

		if err := params.adptPG.UpdateOrder(actionspg.DataOrderAccr(accrResponce)); err != nil {
			return fmt.Errorf("ошибка обновления данных заказа: <%w>", err)
		}
	}

	return nil
}

// Функция запускается один раз при запуске приложения. Выполняет проверку заказов по стутусу NEW и переносит их в очередь на обработку. Возвращает ошибку.
//
// Параметры:
//
// adptPG - адаптер Postgres.
func startUpCheckOrdersByNEW(adptPG actionspg.Postgres) error {

	offset := 0

	for {
		// Чтение номеров по 10
		orders, err := adptPG.GetNewOrderNumbers(offset)
		if err != nil {
			return fmt.Errorf("функция adptPG.GetNewOrderNumbers вернула ошибку: <%w>", err)
		}
		offset += 10

		// Передача в очередь
		for _, order := range orders {

			err := adptPG.AddOrderInQueue(order)
			if err != nil {
				errBase := errors.Unwrap(err)
				if errBase.Error() == ErrOrderExistInQueue { // Если такой номер уже существует в очереди
					continue
				} else {
					return fmt.Errorf("ошибка при сохранении заказа в резервной очереди: <%w>", err)
				}
			}
		}

		// Проверка на финальный запрос
		if len(orders) < 10 {
			break
		}
	}

	return nil
}

// Функция выполняет проверку значений id в таблицах.  Возвращает ошибку.
//
// Параметры:
//
// adptPG - адаптер Postgres.
func checkIDTables(adptPG actionspg.Postgres) error {

	// Запуск проверки
	warningFlagsID, err := adptPG.CheckIDTables()
	if err != nil {
		return fmt.Errorf("функция adptPG.CheckIDTables, вернула ошибку: <%w>", err)
	}

	// Оценка результата
	if warningFlagsID.Users {
		logger.Log.Warn("Предупреждение по ID",
			zap.String("таблица", "users"),
		)
	}
	if warningFlagsID.UserTokens {
		logger.Log.Warn("Предупреждение по ID",
			zap.String("таблица", "user_tokens"),
		)
	}
	if warningFlagsID.Orders {
		logger.Log.Warn("Предупреждение по ID",
			zap.String("таблица", "orders"),
		)
	}
	if warningFlagsID.QueueOrder {
		logger.Log.Warn("Предупреждение по ID",
			zap.String("таблица", "queue_order"),
		)
	}
	if warningFlagsID.Balance {
		logger.Log.Warn("Предупреждение по ID",
			zap.String("таблица", "balance"),
		)
	}
	if warningFlagsID.Withdrawals {
		logger.Log.Warn("Предупреждение по ID",
			zap.String("таблица", "withdrawals"),
		)
	}

	return nil
}
