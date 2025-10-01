package actions

import (
	"database/sql"
	"sync"
	"time"

	actionsaccr "github.com/Part001-R/YaPr-GP-1/internal/adapter/accrual/actions"
	actionspg "github.com/Part001-R/YaPr-GP-1/internal/adapter/postgres/actions"
)

// Данные для регистрации пользователя
type RegisterData struct {
	Login    string
	Password string
}

// Для представления списка заказов
type Order struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual,omitempty"`
	UploadedAt string  `json:"uploaded_at"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type BalanceWithdraw struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type DB struct {
	Ptr *sql.DB
}

// Для приёма ответа от сервиса Accrual
type ResponceAccrual struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

// Для взаимодействия с go рутиной обработки очереди запросов к Accrual
type ChannelsAccrual struct {
	NumbOrder    chan string
	ResponceAccr chan ResponceAccrual
}

// Мьютексы для работы с адаптерами
type mutexes struct {
	register           sync.Mutex
	authentication     sync.Mutex
	addOrder           sync.Mutex
	getOrdersUser      sync.Mutex
	getUserBalance     sync.Mutex
	balanceWithdraw    sync.Mutex
	historyWithdrawals sync.Mutex
}

// Конфигурация сервиса
type ActionsConf struct {
	AdptPG         actionspg.Postgres
	AdptAccr       actionsaccr.Accrual
	ChAccrNewOrder chan string
	mu             mutexes
}

// История вывода
type HistoryWithdrawals struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

// Интерфейсы
type ActionsReg interface {
	RegistrationUser(login, password string) (token string, err error)
}

type ActionsAuth interface {
	AuthenticationUser(login, password string) (string, error)
}

type ActionsAddOrder interface {
	AddOrder(token, order string) error
}

type ActionsGetOrdersUser interface {
	GetOrdersUser(token string) ([]Order, error)
}

type ActionsGetUserBalance interface {
	GetUserBalance(token string) (Balance, error)
}

type ActionsHistoryWithdrawels interface {
	HistoryWithdrawals(token string) ([]HistoryWithdrawals, error)
}

type ActionsBalanceWithdraw interface {
	BalanceWithdraw(token string, dataRx BalanceWithdraw) error
}

type Actions interface {
	ActionsReg
	ActionsAuth
	ActionsAddOrder
	ActionsGetOrdersUser
	ActionsGetUserBalance
	ActionsHistoryWithdrawels
	ActionsBalanceWithdraw
}

// Создание экземпляра
func NewInstServiceActionsT(adprPG actionspg.Postgres, adptAccr actionsaccr.Accrual) *ActionsConf {
	return &ActionsConf{
		AdptPG:         adprPG,
		AdptAccr:       adptAccr,
		ChAccrNewOrder: make(chan string),
		mu: mutexes{
			register:           sync.Mutex{},
			authentication:     sync.Mutex{},
			addOrder:           sync.Mutex{},
			getOrdersUser:      sync.Mutex{},
			getUserBalance:     sync.Mutex{},
			balanceWithdraw:    sync.Mutex{},
			historyWithdrawals: sync.Mutex{},
		},
	}
}

// Создание экземпляра
func NewInstServiceActionsI(params *ActionsConf) Actions {
	return params
}
