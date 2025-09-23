package actions

import (
	"database/sql"
	"sync"
	"time"

	actionsaccr "github.com/Part001-R/YaPr-GP-1/internal/adapter/accrual/actions"
	actionspg "github.com/Part001-R/YaPr-GP-1/internal/adapter/postgres/actions"
)

// Данные для регистрации пользователя
type RegisterDataT struct {
	Login    string
	Password string
}

// Для представления списка заказов
type OrderT struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type BalanceT struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type BalanceWithdrawT struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type DBT struct {
	Ptr *sql.DB
}

// Для приёма ответа от сервиса Accrual
type ResponceAccrualT struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

// Для взаимодействия с go рутиной обработки очереди запросов к Accrual
type ChannelsAccrualT struct {
	NumbOrder    chan string
	ResponceAccr chan ResponceAccrualT
}

// Мьютексы для работы с таблицами БД
type MutexesT struct {
	Register sync.Mutex
}

// Конфигурация сервиса
type ActionsT struct {
	AdptPG         actionspg.PostgresI
	AdptAccr       actionsaccr.AccrualI
	ChAccrNewOrder chan string
}

// История вывода
type HistoryWithdrawalsT struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

// Интерфейсы
type ActionsRegI interface {
	RegistrationUser(login, password string) (string, error)
}

type ActionsAuthI interface {
	AuthenticationUser(login, password string) (string, error)
}

type ActionsAddOrderI interface {
	AddOrder(token, order string) error
}

type ActionsGetOrdersUserI interface {
	GetOrdersUser(token string) ([]OrderT, error)
}

type ActionsGetUserBalanceI interface {
	GetUserBalance(token string) (BalanceT, error)
}

type ActionsHistoryWithdrawelsI interface {
	HistoryWithdrawels(token string) ([]HistoryWithdrawalsT, error)
}

type ActionsBalanceWithdrawI interface {
	BalanceWithdraw(token string, dataRx BalanceWithdrawT) error
}

type ActionsI interface {
	ActionsRegI
	ActionsAuthI
	ActionsAddOrderI
	ActionsGetOrdersUserI
	ActionsGetUserBalanceI
	ActionsHistoryWithdrawelsI
	ActionsBalanceWithdrawI
}

// Создание экземпляра T
func NewInstServiceActionsT(adprPG actionspg.PostgresI, adptAccr actionsaccr.AccrualI) *ActionsT {
	return &ActionsT{
		AdptPG:         adprPG,
		AdptAccr:       adptAccr,
		ChAccrNewOrder: make(chan string),
	}
}

// Создание экземпляра I
func NewInstServiceActionsI(params *ActionsT) ActionsI {
	return params
}
