package actions

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"go.uber.org/zap"
)

// Функция выполняет вызов метода адаптера Postgres с регистрацией пользователя. Возвращается токен и ошибка.
//
// Параметры:
// login - логин.
// password - пароль.
func (a *ActionsConf) RegistrationUser(login, password string) (token string, err error) {

	a.mu.register.Lock()
	defer a.mu.register.Unlock()

	// Проверка аргументов
	if login == "" || password == "" {
		return "", errors.New("в одном из аргументов пустое значение")
	}

	// Логика
	//
	// Начало транзакции
	tx, err := a.AdptPG.BeginTx()
	if err != nil {
		return "", fmt.Errorf("ошибка начала транзакции: <%w>", err)
	}

	defer func() {
		if err != nil {
			rbErr := tx.Rollback()
			if rbErr != nil {
				err = fmt.Errorf("ошибка tx.Rollback при регистрации пользователя: <%w>", rbErr)
			}
		}
	}()

	// Регистрация пользователя
	userID, err := a.AdptPG.RegisterUser(tx, login, password)
	if err != nil {
		logger.Log.Error("ошибка регистрации пользователя",
			zap.String("err", err.Error()),
		)
		return "", fmt.Errorf("ошибка регистрации пользователя: <%w>", err)
	}

	// Создание токена
	token, err = a.AdptPG.CreateUpdateToken(tx, userID)
	if err != nil {
		logger.Log.Error("ошибка создания токена",
			zap.String("err", err.Error()),
		)
		return "", fmt.Errorf("ошибка создания токена: <%w>", err)
	}

	// Создание баланса пользователя
	err = a.AdptPG.CreateUserBalance(tx, userID)
	if err != nil {
		logger.Log.Error("ошибка создания баланса пользователя",
			zap.String("err", err.Error()),
		)
		return "", fmt.Errorf("ошибка создания баланса пользователя: <%w>", err)
	}

	// Подтверждение изменений
	err = a.AdptPG.CommitTx(tx)
	if err != nil {
		logger.Log.Error("ошибка подтверждения транзакции",
			zap.String("err", err.Error()),
		)
		return "", fmt.Errorf("ошибка подтверждения транзакции: <%w>", err)
	}

	// Результат
	return token, nil
}

// Функция выполняет вызов метода адаптера Postgres с аутентификацией пользователя. Возвращается токен и ошибка.
//
// Параметры:
// login - логин.
// password - пароль.
func (a *ActionsConf) AuthenticationUser(login, password string) (string, error) {

	a.mu.authentication.Lock()
	defer a.mu.authentication.Unlock()

	// Проверка аргументов
	if login == "" || password == "" {
		return "", errors.New("в одном из аргументов пустое значение")
	}

	// Логика
	token, err := a.AdptPG.AuthenticationUser(login, password)
	if err != nil {
		return "", fmt.Errorf("ошибка при аутентификации пользователя: <%w>", err)
	}

	// Результат
	return token, nil
}

// Функция выполняет проверку номера заказа и вызывает функции адаптера Postgres. Возвращается ошибка.
//
// Параметры:
// token - токен.
// order - заказ.
func (a *ActionsConf) AddOrder(token, order string) error {

	a.mu.addOrder.Lock()
	defer a.mu.addOrder.Unlock()

	// Проверка аргументов
	if token == "" {
		return errors.New("в аргументе token нет содержимого")
	}
	if order == "" {
		return errors.New("в аргументе order нет содержимого")
	}

	// Валидация номера заказа алгоритмом Луна
	isValid, err := isValidByLuhn(order)
	if err != nil {
		return fmt.Errorf("функция validationByLuna, вернула ошибку: <%w>", err)
	}
	if !isValid {
		return errors.New("неверный формат номера заказа")
	}

	// Логика
	//
	// Получение id пользователя.
	// При выполнении, проверяется аутентификация и авторизация
	userID, err := a.AdptPG.GetUserIDByToken(token)
	if err != nil {
		return fmt.Errorf("функция GetUserIDByToken, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Добавление заказа
	if err := a.AdptPG.AddOrder(order, userID); err != nil {
		return fmt.Errorf("функция AdapterPGAddOrder, адаптера AdptPG,  вернула ошибку: <%w>", err)
	}

	// Передача номера заказа в канал, для последующего взаимодействия с Accrual.
	// Данные из канала принимает go рутина (runAccrual).
	// Запускается в слое service, в функции server.
	select {
	case a.ChAccrNewOrder <- order:
	case <-time.After(3 * time.Second):
		logger.Log.Error("отправленные данные в канал не прочитаны за отведённое время")
		return errors.New("отправленные данные в канал не прочитаны за отведённое время")
	}

	// Результат
	return nil
}

// Функция вызывает функцию адаптера Postgres по получению списка заказов. Возвращает список заказов и ошибку.
//
// Параметры:
//
// token - токен.
func (a *ActionsConf) GetOrdersUser(token string) ([]Order, error) {

	a.mu.getOrdersUser.Lock()
	defer a.mu.getOrdersUser.Unlock()

	// Проверка аргументов
	if token == "" {
		return nil, errors.New("в аргументе token нет содержимого")
	}

	// Логика
	ordersRx, err := a.AdptPG.GetOrdersUser(token)
	if err != nil {
		return nil, fmt.Errorf("функция GetOrdersUser вернула ошибку: <%w>", err)
	}
	if len(ordersRx) == 0 {
		return nil, errors.New("нет данных для ответа")
	}

	// Перенос принятых данных
	ordersTx := make([]Order, 0)
	var el Order
	for _, v := range ordersRx {
		el.Accrual = v.Accrual
		el.Number = v.Number
		el.Status = v.Status
		el.UploadedAt = v.UploadedAt

		ordersTx = append(ordersTx, el)
	}

	// Результат
	return ordersTx, nil
}

// Функция вызывает функцию адаптера Postgres по получению баланса. Возвращает информацию по балансу пользователя и ошибку.
//
// Параметры:
//
// token - токен.
func (a *ActionsConf) GetUserBalance(token string) (Balance, error) {

	a.mu.getUserBalance.Lock()
	defer a.mu.getUserBalance.Unlock()

	// Проверка аргументов
	if token == "" {
		return Balance{}, errors.New("в аргументе token нет содержимого")
	}

	// Логика
	//
	// Получение id пользователя.
	// Проверяется аутентификация и авторизация.
	userID, err := a.AdptPG.GetUserIDByToken(token)
	if err != nil {
		return Balance{}, fmt.Errorf("функция GetUserIDByToken, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Получение текущего баланса пользоателя.
	balanceRx, err := a.AdptPG.GetUserBalance(userID)
	if err != nil {
		return Balance{}, fmt.Errorf("функция GetUserBalance, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Перенос принятых данных
	var curBalance Balance
	curBalance.Current = balanceRx.Current
	curBalance.Withdrawn = balanceRx.Withdrawn

	// Возврат
	return curBalance, nil
}

// Функция с действиями по списанию средств. Возвращает ошибку.
//
// Параметры:
//
// token - токен.
func (a *ActionsConf) BalanceWithdraw(token string, dataRx BalanceWithdraw) (err error) {

	a.mu.balanceWithdraw.Lock()
	defer a.mu.balanceWithdraw.Unlock()

	// Проверка аргументов
	if token == "" {
		return errors.New("в аргументе token нет содержимого")
	}
	if dataRx.Order == "" {
		return errors.New("неверный номер заказа")
	}
	if dataRx.Sum <= 0 {
		return errors.New("неверный номер заказа") // по ТЗ нет кода при этом условии. Использую - неверный номер заказа
	}

	// Проверка номера заказа
	isValid, err := isValidByLuhn(dataRx.Order)
	if err != nil {
		return fmt.Errorf("ошибка в функции validationByLuna: <%w>", err)
	}
	if !isValid {
		return errors.New("неверный номер заказа")
	}

	// Логика
	//
	// Получение id пользователя.
	// При выполнении, проверяется аутентификация и авторизация
	userID, err := a.AdptPG.GetUserIDByToken(token)
	if err != nil {
		return fmt.Errorf("функция GetUserIDByToken, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Получение текущего баланса пользователя
	curBalance, err := a.AdptPG.GetUserBalance(userID)
	if err != nil {
		return fmt.Errorf("функция GetUserBalance, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Проверка, что на счету пользователя достаточно баллов для списания
	ok, err := isPossibilityWithdraw(dataRx.Sum, curBalance.Current)
	if err != nil {
		return fmt.Errorf("функция isPossibilityWithdraw вернула ошибку: <%w>", err)
	}
	if !ok {
		return errors.New("на счету недостаточно средств")
	}

	// Транзакция
	tx, err := a.AdptPG.BeginTx()
	if err != nil {
		return fmt.Errorf("функция BeginTx, адаптера AdptPG, вернула ошибку: <%w>", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				logger.Log.Error("Ошибка tx.Rollback",
					zap.String("err", rbErr.Error()),
				)
				err = rbErr
			}
		}
	}()

	// Списание баллов
	err = a.AdptPG.DoWithdrawTx(tx, userID, dataRx.Sum, curBalance, dataRx.Order)
	if err != nil {
		logger.Log.Error("Функция DoWithdraw вернула ошибку",
			zap.String("err", err.Error()),
		)
		return fmt.Errorf("функция DoWithdraw вернула ошибку: <%w>", err)
	}

	// Создание заказа
	err = a.AdptPG.AddOrderTx(tx, dataRx.Order, userID)
	if err != nil {
		logger.Log.Error("Функция AddOrder, адаптера AdptPG, вернула ошибку",
			zap.String("err", err.Error()),
		)
		return fmt.Errorf("функция AddOrder, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Подтверждение транзакции
	err = a.AdptPG.CommitTx(tx)
	if err != nil {
		logger.Log.Error("Функция CommitTx, адаптера AdptPG, вернула ошибку",
			zap.String("err", err.Error()),
		)
		return fmt.Errorf("функция CommitTx, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Передача номера заказа в канал, для последующего взаимодействия с Accrual.
	// Данные из канала принимает go рутина (runAccrual).
	// Запускается в слое service, в функции server.
	select {
	case a.ChAccrNewOrder <- dataRx.Order:
	case <-time.After(3 * time.Second):
		logger.Log.Error("отправленные данные в канал не прочитаны за отведённое время")
		return errors.New("отправленные данные в канал не прочитаны за отведённое время")
	}

	return nil
}

// Функция получает историю вывода пользователя. Возвращает историю вывода и ошибку.
//
// Параметры:
//
// token - токен пользователя.
func (a *ActionsConf) HistoryWithdrawals(token string) ([]HistoryWithdrawals, error) {

	a.mu.historyWithdrawals.Lock()
	defer a.mu.historyWithdrawals.Unlock()

	// Проверка аргументов
	if token == "" {
		return nil, errors.New("в аргументе token нет содержимого")
	}

	// Вызов функции адаптера PG
	history, err := a.AdptPG.HistoryWithrawals(token)
	if err != nil {
		return nil, fmt.Errorf("функция HistoryWithrawals, адаптера PG, вернула ошибку: <%w>", err)
	}

	// Перенос данных
	copyHistory := make([]HistoryWithdrawals, 0)

	for _, v := range history {
		var el HistoryWithdrawals
		el.Order = v.Order
		el.Sum = v.Sum
		el.ProcessedAt = v.ProcessedAt

		copyHistory = append(copyHistory, el)
	}

	// Ответ
	return copyHistory, nil
}

// Реализация алгоритма Луна. Возвращает true - если валидно и ошибку.
//
// Параметры:
//
// number - номер для проверки.
func isValidByLuhn(number string) (bool, error) {

	// Проверка аргументов
	if len(number) == 0 {
		return false, errors.New("в аргументе number нет содержимого")
	}

	// Логика
	sum := 0
	even := false

	for i := len(number) - 1; i >= 0; i-- {

		digit, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return false, errors.New("неверный формат номера заказа")
		}

		// Обработка четных позиций
		if even {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		even = !even
	}

	// Результат
	return sum%10 == 0, nil
}

// Функция выполняет проверку баланса пользователя для списания. Возвращает true - действие доступно и ошибку.
//
// Параметры:
//
// forWithdraw - сумма для списания.
// currentBalance - текущий балланс.
func isPossibilityWithdraw(forWithdraw float64, currentBalance float64) (bool, error) {

	// Проверка аргументов
	if forWithdraw <= 0 {
		return false, errors.New("недопустимое содержимое в аргументе forWithdraw")
	}
	if currentBalance < 0 {
		return false, errors.New("недопустимое содержимое в аргументе currentBalance")
	}

	// Логика
	currentBalance *= 100
	forWithdraw *= 100

	if forWithdraw > currentBalance {
		return false, nil
	}

	return true, nil
}
