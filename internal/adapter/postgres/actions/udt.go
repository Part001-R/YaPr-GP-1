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
type PostgresConf struct {
	PtrDB     *sql.DB
	muBalance sync.Mutex
}

type Order struct {
	Number     string
	Status     string
	Accrual    float64
	UploadedAt string
}

type WarningFlagsID struct {
	Users       bool
	UserTokens  bool
	Orders      bool
	QueueOrder  bool
	Balance     bool
	Withdrawals bool
}

// Данные по заказу принятые от Accrual
type DataOrderAccr struct {
	Order   string
	Status  string
	Accrual float64
}

type BalanceWithdraw struct {
	Order string
	Sum   float64
}

type HistoryWithdrawals struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

type Balance struct {
	Current   float64
	Withdrawn float64
}

// Интерфейсы
type RegistrationUser interface {
	RegisterUser(tx *sql.Tx, login, password string) (int64, error)
}

type AuthenticationUser interface {
	AuthenticationUser(login, password string) (string, error)
}

type AddOrder interface {
	AddOrder(order string, userID int64) error
}

type AddOrderTx interface {
	AddOrderTx(tx *sql.Tx, order string, userID int64) error
}

type GetOrdersUser interface {
	GetOrdersUser(token string) (orders []Order, err error)
}

type GetUserBalance interface {
	GetUserBalance(userID int64) (Balance, error)
}

type HistoryWithrawals interface {
	HistoryWithrawals(token string) (hw []HistoryWithdrawals, err error)
}

type CreateUserBalance interface {
	CreateUserBalance(tx *sql.Tx, userID int64) error
}

type GetUserIDByToken interface {
	GetUserIDByToken(token string) (int64, error)
}

type DoWithdrawTx interface {
	DoWithdrawTx(tx *sql.Tx, userID int64, sumWithdraw float64, curBalance Balance, order string) error
}

type UpdateOrder interface {
	UpdateOrder(data DataOrderAccr) error
}

type AddOrderInQueue interface {
	AddOrderInQueue(orderNumber string) error
}

type GetOrdersInQueue interface {
	GetOrdersInQueue() ([]string, error)
}

type UpdateOrderStatus interface {
	UpdateOrderStatus(data DataOrderAccr) error
}

type CreateUpdateToken interface {
	CreateUpdateToken(tx *sql.Tx, id int64) (string, error)
}

type BeginTx interface {
	BeginTx() (*sql.Tx, error)
}

type CommitTx interface {
	CommitTx(tx *sql.Tx) error
}

type GetNewOrderNumbers interface {
	GetNewOrderNumbers(offset int) ([]string, error)
}

type CheckIDTables interface {
	CheckIDTables() (WarningFlagsID, error)
}

type Postgres interface {
	RegistrationUser
	AuthenticationUser
	AddOrder
	AddOrderTx
	GetOrdersUser
	GetUserBalance
	HistoryWithrawals
	CreateUserBalance
	GetUserIDByToken
	DoWithdrawTx
	UpdateOrder
	AddOrderInQueue
	GetOrdersInQueue
	UpdateOrderStatus
	CreateUpdateToken
	BeginTx
	CommitTx
	GetNewOrderNumbers
	CheckIDTables
}

// Создание экземпляра адаптера
func NewInstAdapterPostgres(db *sql.DB) Postgres {
	return &PostgresConf{
		PtrDB:     db,
		muBalance: sync.Mutex{},
	}
}
