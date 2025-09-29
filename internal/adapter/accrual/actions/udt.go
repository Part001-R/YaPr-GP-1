package actionsaccr

const (
	ErrNotExistOrder     = "заказ не зарегистрирован в системе расчёта"
	ErrTooManyRequests   = "превышено количество запросов к сервису"
	ErrInternalServerErr = "внутренняя ошибка сервера"
)

type OrderDataRx struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type AccrualConf struct {
	Address string
}

type GetOrderInfo interface {
	GetOrderInfo(order string) (OrderDataRx, error)
}

type Accrual interface {
	GetOrderInfo
}

// Создание экземпляра адаптера
func NewInstAdapterAccrual(address string) Accrual {
	return &AccrualConf{
		Address: address,
	}
}
