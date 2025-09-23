package actionspg

import (
	"database/sql"
	"sync"
	"time"
)

const (
	errConflictToken        = `pq: duplicate key value violates unique constraint "user_tokens_token_key"`
	errDuplicateOrderByUser = `pq: duplicate key value violates unique constraint "orders_user_id_order_number_key"`
	errNumbOrderBusy        = `pq: duplicate key value violates unique constraint "orders_order_number_key"`
)

// Типы данных
type MutexesT struct {
	Registration sync.RWMutex
}

type PostgresT struct {
	PtrDB *sql.DB
}

type OrderT struct {
	Number     string
	Status     string
	Accrual    float64
	UploadedAt string
}

// Данные по заказу принятые от Accrual
type DataOrderAccr struct {
	Order   string
	Status  string
	Accrual float64
}

type BalanceWithdrawT struct {
	Order string
	Sum   float64
}

type HistoryWithdrawalsT struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

type BalanceT struct {
	Current   float64
	Withdrawn float64
}

// Интерфейсы
type RegistrationUserDBI interface {
	RegisterUser(login, password string) (int64, error)
}

type AuthenticationUserDBI interface {
	AuthenticationUser(login, password string) (string, error)
}

type AddOrderDBI interface {
	AddOrder(order string, userID int64) error
}

type GetOrdersUserDBI interface {
	GetOrdersUser(token string) (orders []OrderT, err error)
}

type GetUserBalanceDBI interface {
	GetUserBalance(userID int64) (BalanceT, error)
}

type HistoryWithrawalsDBI interface {
	HistoryWithrawals(token string) ([]HistoryWithdrawalsT, error)
}

type CreateUserBalanceDBI interface {
	CreateUserBalance(userID int64) error
}

type GetUserIDByTokenDBI interface {
	GetUserIDByToken(token string) (int64, error)
}

type DoWithdrawDBI interface {
	DoWithdraw(userID int64, sumWithdraw float64, curBalance BalanceT, order string) error
}

type UpdateOrderDBI interface {
	UpdateOrder(data DataOrderAccr) error
}

type AddOrderInQueueDBI interface {
	AddOrderInQueue(orderNumber string) error
}

type GetOrdersInQueueDBI interface {
	GetOrdersInQueue() ([]string, error)
}

type UpdateOrderStatusDBI interface {
	UpdateOrderStatus(data DataOrderAccr) error
}

type CreateUpdateTokenDBI interface {
	CreateUpdateToken(id int64) (string, error)
}

type PostgresI interface {
	RegistrationUserDBI
	AuthenticationUserDBI
	AddOrderDBI
	GetOrdersUserDBI
	GetUserBalanceDBI
	HistoryWithrawalsDBI
	CreateUserBalanceDBI
	GetUserIDByTokenDBI
	DoWithdrawDBI
	UpdateOrderDBI
	AddOrderInQueueDBI
	GetOrdersInQueueDBI
	UpdateOrderStatusDBI
	CreateUpdateTokenDBI
}

// Создание экземпляра адаптера
func NewInstAdapterPostgres(db *sql.DB) PostgresI {
	return &PostgresT{
		PtrDB: db,
	}
}
