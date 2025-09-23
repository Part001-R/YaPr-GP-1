package service

import (
	"database/sql"
	"os"
	"sync"
	"time"

	actionsaccr "github.com/Part001-R/YaPr-GP-1/internal/adapter/accrual/actions"
	actionspg "github.com/Part001-R/YaPr-GP-1/internal/adapter/postgres/actions"
	"github.com/Part001-R/YaPr-GP-1/internal/controller"
)

const (
	ErrAccrNotExistOrder      = "заказ не зарегистрирован в системе расчёта"
	ErrAccrTooManyRequests    = "превышено количество запросов к сервису"
	ErrAccrInternalServerErr  = "внутренняя ошибка сервера"
	ErrOrderExistInQueue      = "pq: duplicate key value violates unique constraint \"queue_order_order_number_key\""
	ErrTimeoutResponceAccrual = "Client.Timeout exceeded while awaiting headers"
)

// Данные для регистрации пользователя
type RegisterDataT struct {
	Login    string
	Password string
}

type DBT struct {
	Ptr       *sql.DB
	FuncClose func()
}

// Мьютексы для работы с таблицами БД
type MutexesT struct {
	Register sync.Mutex
}

type ChAccrT struct {
	NewOrderNumb *chan string
}

// Конфигурация сервиса
type ServiceT struct {
	DB     DBT
	Mtx    MutexesT
	Cntr   *controller.ControllerT
	ChAccr ChAccrT
}

// Интерфейс регистрации пользователя
type ServiceIntRegI interface {
	RegistrationUser(login, password string) (token string, err error)
}

// Интерфейс сервиса
type ServiceI interface {
	ServiceIntRegI
}

// Причины остановки приложения
type checkReasonStopT struct {
	chCntrErr chan error // остановка контроллера
	chAccrErr chan error // остановка обработчика Accrual
	sigSys    chan os.Signal
}

// Для обработки заказов
type ProcessingOrderT struct {
	tmr                   *time.Timer // для формирования выдержки при коде 429
	newOrdNumb            string
	needStoreByFreq       bool // сохранить в очереди из-за - частые запросы
	needStoreByErr        bool // сохранить в очереди из-за - ошибка в Accrual
	needStoreByNotReg     bool // сохранить в очереди из-за - заказ не зарегистирован
	needStoreByREGISTERED bool // сохранить в очереди из-за - заказ зарегистрирован, но вознаграждение не рассчитано
	needStoreByPROCESSING bool // сохранить в очереди из-за - расчёт начисления в процессе
	adptAccr              actionsaccr.AccrualI
	adptPG                actionspg.PostgresI
}

func NewInstService(db *sql.DB, chRxAccr chan string, fc func(), ctrl *controller.ControllerT) *ServiceT {
	return &ServiceT{
		DB: DBT{
			Ptr:       db,
			FuncClose: fc,
		},
		Mtx: MutexesT{
			Register: sync.Mutex{},
		},
		Cntr: ctrl,
		ChAccr: ChAccrT{
			NewOrderNumb: &chRxAccr,
		},
	}
}
