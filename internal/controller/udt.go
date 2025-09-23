package controller

import (
	"sync"
	"time"

	"github.com/Part001-R/YaPr-GP-1/internal/service/actions"
	"github.com/Part001-R/YaPr-GP-1/internal/utils/flags"
)

const (
	errUserExist          = "pq: duplicate key value violates unique constraint \"users_user_name_key\""
	errPairLoginPassword  = "нет соответствия пары логи-пароль"
	errUserNotFound       = "пользователь не найден"
	errOrderExist         = "номер заказа уже был загружен этим пользователем"
	errNoAuthentication   = "пользователь не аутентифицирован"
	errNoAuthorization    = "пользователь не авторизован"
	errOrderBusy          = "номер заказа уже был загружен другим пользователем"
	errOrderFormat        = "неверный формат номера заказа"
	errOrdersNoContent    = "нет данных для ответа"
	errNotFoundBalance    = "данные баланса пользователя не найдены"
	errNoWithdrawels      = "нет ни одного списания"
	errNotEnoughtBalance  = "на счету недостаточно средств"
	errIncorrectOrderNumb = "неверный номер заказа"
)

// Для приёма данных регистрации
type RegisterRxT struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Для приёма данных логина
type LoginRxT struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type MtxT struct {
	Register              sync.Mutex
	Login                 sync.Mutex
	AddOrder              sync.Mutex
	GetOrdersUser         sync.Mutex
	GetUserBalance        sync.Mutex
	BalanceWithdraw       sync.Mutex
	GetHistoryWithdrawals sync.Mutex
}

type ControllerT struct {
	Flags   flags.FlagsT
	ServAct actions.ActionsI
	Mtx     MtxT
}

// Для истории выводов
type WithdrawalResponse struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

// Для истории выводов (локальная копия)
type HistoryWithdrawalsT struct {
	Order       string
	Sum         int64
	ProcessedAt time.Time
}

func NewInstController(fl flags.FlagsT, servAct actions.ActionsI) *ControllerT {
	return &ControllerT{
		Flags:   fl,
		ServAct: servAct,
		Mtx: MtxT{
			Register:              sync.Mutex{},
			Login:                 sync.Mutex{},
			AddOrder:              sync.Mutex{},
			GetOrdersUser:         sync.Mutex{},
			GetUserBalance:        sync.Mutex{},
			BalanceWithdraw:       sync.Mutex{},
			GetHistoryWithdrawals: sync.Mutex{},
		},
	}
}
