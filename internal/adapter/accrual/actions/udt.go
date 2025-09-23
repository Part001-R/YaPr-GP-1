package actionsaccr

const (
	ErrNotExistOrder     = "заказ не зарегистрирован в системе расчёта"
	ErrTooManyRequests   = "превышено количество запросов к сервису"
	ErrInternalServerErr = "внутренняя ошибка сервера"
)

type OrderDataRxT struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type AccrualT struct {
	Address string
}

type GetOrderInfoI interface {
	GetOrderInfo(order string) (OrderDataRxT, error)
}

type AccrualI interface {
	GetOrderInfoI
}

// Создание экземпляра адаптера
func NewInstAdapterAccrual(address string) AccrualI {
	return &AccrualT{
		Address: address,
	}
}
