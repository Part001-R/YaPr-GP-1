package actions

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

// Функция выполняет вызов метода адаптера Postgres с регистрацией пользователя. Возвращается токен и ошибка.
//
// Параметры:
// login - логин.
// password - пароль.
func (a *ActionsT) RegistrationUser(login, password string) (string, error) {

	// Проверка аргументов
	if login == "" || password == "" {
		return "", errors.New("в одном из аргументов пустое значение")
	}

	// Логика
	//
	// Регистрация пользователя
	userID, err := a.AdptPG.RegisterUser(login, password)
	if err != nil {
		return "", fmt.Errorf("ошибка регистрации пользователя: <%w>", err)
	}

	// Создание токена
	token, err := a.AdptPG.CreateUpdateToken(userID)
	if err != nil {
		return "", fmt.Errorf("ошибка создания токена: <%w>", err)
	}

	// Создание баланса пользователя
	if err := a.AdptPG.CreateUserBalance(userID); err != nil {
		return "", fmt.Errorf("ошибка создания баланса пользователя: <%w>", err)
	}

	// Результат
	return token, nil
}

// Функция выполняет вызов метода адаптера Postgres с аутентификацией пользователя. Возвращается токен и ошибка.
//
// Параметры:
// login - логин.
// password - пароль.
func (a *ActionsT) AuthenticationUser(login, password string) (string, error) {

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
func (a *ActionsT) AddOrder(token, order string) error {

	// Проверка аргументов
	if token == "" {
		return errors.New("в аргументе token нет содержимого")
	}
	if order == "" {
		return errors.New("в аргументе order нет содержимого")
	}

	// Валидация номера заказа алгоритмом Луна
	isValid, err := validationByLuna(order)
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
func (a *ActionsT) GetOrdersUser(token string) ([]OrderT, error) {

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
	ordersTx := make([]OrderT, 0)
	var el OrderT
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
func (a *ActionsT) GetUserBalance(token string) (BalanceT, error) {

	// Проверка аргументов
	if token == "" {
		return BalanceT{}, errors.New("в аргументе token нет содержимого")
	}

	// Логика
	//
	// Получение id пользователя.
	// Проверяется аутентификация и авторизация.
	userID, err := a.AdptPG.GetUserIDByToken(token)
	if err != nil {
		return BalanceT{}, fmt.Errorf("функция GetUserIDByToken, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Получение текущего баланса пользоателя.
	balanceRx, err := a.AdptPG.GetUserBalance(userID)
	if err != nil {
		return BalanceT{}, fmt.Errorf("функция GetUserBalance, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Перенос принятых данных
	var curBalance BalanceT
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
func (a *ActionsT) BalanceWithdraw(token string, dataRx BalanceWithdrawT) error {

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
	isValid, err := validationByLuna(dataRx.Order)
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

	// Списание баллов
	if err := a.AdptPG.DoWithdraw(userID, dataRx.Sum, curBalance, dataRx.Order); err != nil {
		return fmt.Errorf("функция DoWithdraw вернула ошибку: <%w>", err)
	}

	// Создание заказа
	if err := a.AdptPG.AddOrder(dataRx.Order, userID); err != nil {
		return fmt.Errorf("функция AddOrder, адаптера AdptPG, вернула ошибку: <%w>", err)
	}

	// Передача номера заказа в канал, для последующего взаимодействия с Accrual.
	// Данные из канала принимает go рутина (runAccrual).
	// Запускается в слое service, в функции server.
	select {
	case a.ChAccrNewOrder <- dataRx.Order:
	case <-time.After(3 * time.Second):
		return errors.New("отправленные данные в канал не прочитаны за отведённое время")
	}

	return nil
}

// Функция для запуска в go рутине. Выполняет взаимодействие с Accrual.
//
// Параметры:
//
// ch - каналы для взаимодействия с go рутиной.
// addrAccr - адрес Accrual сервиса.
// chErr - канал возврата ошибки.
func (a *ActionsT) RunQueueAccrual(ch ChannelsAccrualT, addrAccr string, chErr chan error) {

	// Проверка аргументов
	if chErr == nil {
		log.Fatalf("Нет канала для chErr. Работа прервана.")
	}
	if ch.ResponceAccr == nil {
		chErr <- errors.New("нет канала передачи ответа от Accrual")
		return
	}
	if ch.NumbOrder == nil {
		chErr <- errors.New("нет канала приёма номера заказа, для отправки в Accrual")
		return
	}
	if addrAccr == "" {
		chErr <- errors.New("нет содержимого в addrAccr ")
		return
	}

}

// Функция получает историю вывода пользователя. Возвращает историю вывода и ошибку.
//
// Параметры:
//
// token - токен пользователя.
func (a *ActionsT) HistoryWithdrawels(token string) ([]HistoryWithdrawalsT, error) {

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
	copyHistory := make([]HistoryWithdrawalsT, 0)

	for _, v := range history {
		var el HistoryWithdrawalsT
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
func validationByLuna(number string) (bool, error) {

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
