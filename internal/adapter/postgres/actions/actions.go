package actionspg

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"go.uber.org/zap"
)

// Функция добавляет пользователя в таблицу пользователей и добавляет токен в таблицу токенов. Возвращает токен и ошибку.
//
// Параметры:
//
// login - имя пользователя.
// password - пароль.
func (a *PostgresT) RegisterUser(login, password string) (int, error) {

	// Проверка аргументов
	if a.PtrDB == nil {
		return 0, errors.New("нет указателя на БД")
	}
	if login == "" {
		return 0, errors.New("в аргументе login нет содержимого")
	}
	if password == "" {
		return 0, errors.New("в аргументе password нет содержимого")
	}

	// Начало транзакции
	tx, err := a.PtrDB.Begin()
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции: <%w>", err)
	}

	// Добавление пользователя
	newUserID, err := addUserTx(tx, login, password)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("ошибка при добавлении пользователя: <%w>", err)
	}

	// Подтверждение транзакции
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("ошибка при подтверждении транзакции: <%w>", err)
	}

	// Результат
	return newUserID, nil
}

// Функция создаёт счёт пользователя, при его регистрации. Возвращает ошибку.
//
// Параметры:
//
// userID - id пользователя.
func (a *PostgresT) CreateUserBalance(userID int) error {

	// Проверка аргументов
	if a.PtrDB == nil {
		return errors.New("нет указателя на БД")
	}
	if userID < 1 {
		return errors.New("недопустимое содержимое в аргумете userID")
	}

	// Логика
	query := `
        INSERT INTO balance (user_id, accrual, withdrawn)
        VALUES ($1, $2, $3)
    `
	result, err := a.PtrDB.Exec(query, userID, 0, 0)
	if err != nil {
		return fmt.Errorf("ошибка при создании баланса: %w", err)
	}

	id, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка в получении id добавленной записи: %w", err)
	}

	if id < 1 {
		return errors.New("счёт не создан")
	}

	return nil
}

// Функция выполняет аутентификацию пользователя. Возвращает токен и ошибку.
//
// Параметры:
//
// login - имя пользователя.
// password - пароль.
func (a *PostgresT) AuthenticationUser(login, password string) (string, error) {

	// Проверка аргументов
	if a.PtrDB == nil {
		return "", errors.New("нет указателя на БД")
	}
	if login == "" || password == "" {
		return "", errors.New("в одном из аргументов нет содержимого")
	}

	// Чтение имени и id пользователя по паролю
	userPwd, id, err := getPasswordByUserNameDB(a.PtrDB, login)
	if err != nil {
		return "", fmt.Errorf("функция getUserNameByPassword вернула ошибку: <%w>", err)
	}

	// Генерация хеша по принятому пароля в аутентификации
	hashRxPwd, err := hashString(password)
	if err != nil {
		return "", errors.New("ошибка генерации хеша")
	}

	if userPwd != hashRxPwd {
		return "", errors.New("нет соответствия пары логи-пароль")
	}

	// Обновление токена
	newToken, err := updateTokenDB(a.PtrDB, id)
	if err != nil {
		return "", fmt.Errorf("функция updateTokenDB вернула ошибку: <%w>", err)
	}

	// Результат
	return newToken, err
}

// Функция выполняет добавление номера заказа. Возвращает код выполнения и ошибку.
//
// Параметры:
//
// order - номер заказа.
// token - токен.
func (a *PostgresT) AddOrder(order string, userID int64) error {

	// Проверка аргументов
	if a.PtrDB == nil {
		return errors.New("нет указателя на БД")
	}
	if order == "" {
		return errors.New("в аргументе order нет содержимого")
	}
	if userID <= 0 {
		return errors.New("в аргументе userID недопустимое значение")
	}

	// Логика
	if err := addUserOrderDB(a.PtrDB, userID, order); err != nil {
		return fmt.Errorf("функция addUserOrderDB вернула ошибку: <%w>", err)
	}

	// Результат
	return nil
}

// Функция формирует массив звказов по токену. Возвращает массив заказов и ошибку.
//
// Параметры:
//
// token - токен.
func (a *PostgresT) GetOrdersUser(token string) (orders []OrderT, err error) {

	// Проверка аргументов
	if a.PtrDB == nil {
		return nil, errors.New("нет указателя на БД")
	}
	if token == "" {
		return nil, errors.New("в аргументе token нет содержимого")
	}

	// Логика
	userID, err := getIdByToken(a.PtrDB, token)
	if err != nil {
		return nil, fmt.Errorf("функция getIdByToken вернула ошибку: <%w>", err)
	}
	orders, err = getOrdersUser(a.PtrDB, userID)
	if err != nil {
		return nil, fmt.Errorf("функция getOrdersUser вернула ошибку: <%w>", err)
	}

	// Результат
	return orders, nil
}

// Функция предоставляет информацию по балансу пользователя. Возвращает баланс и ошибку.
//
// Параметры:
//
// userID - шв пользователя.
func (a *PostgresT) GetUserBalance(userID int64) (BalanceT, error) {

	// Проверка аргументов
	if a.PtrDB == nil {
		return BalanceT{}, errors.New("нет указателя на БД")
	}
	if userID <= 0 {
		return BalanceT{}, errors.New("недопустимое значение в userID")
	}

	// Логика
	balance, err := getUserBalance(a.PtrDB, userID)
	if err != nil {
		return BalanceT{}, fmt.Errorf("функция getUserBalance вернула ошибку: <%w>", err)
	}

	// Возврат
	return balance, err
}

// Функция предоставляет id пользователя по принятому токену. Возвращает id пользователя и ошибку.
//
// Параметры:
//
// token - токен.
func (a *PostgresT) GetUserIDByToken(token string) (int64, error) {

	// Проверка аргументов
	if token == "" {
		return 0, errors.New("в аргументе token нет содержимого")
	}

	// Логика
	userID, err := getIdByToken(a.PtrDB, token)
	if err != nil {
		return 0, fmt.Errorf("функция getIdByToken вернула ошибку: <%w>", err)
	}

	// Возврат
	return userID, nil
}

// Функция выполняет списание баллов пользователя. Возвращает ошибку.
func (a *PostgresT) DoWithdraw(userID int64, sumWithdraw float64, curBalance BalanceT, order string) error {

	// Проверка аргументов
	if userID <= 0 {
		return errors.New("в аргументе userID недопустимое зночение")
	}
	if sumWithdraw <= 0 {
		return errors.New("в аргументе sumWithdraw недопустимое зночение")
	}
	if curBalance.Current <= 0 {
		return errors.New("в аргументе curBalance.Current недопустимое зночение")
	}
	if curBalance.Withdrawn < 0 {
		return errors.New("в аргументе curBalance.Withdrawn недопустимое зночение")
	}
	if order == "" {
		return errors.New("в аргументе order недопустимое зночение")
	}

	// Транзакция
	tx, err := a.PtrDB.Begin()
	if err != nil {
		return errors.New("ошибка при создании транзакции")
	}

	// Выполнение списания
	if err := balanceWithdrawTx(tx, userID, sumWithdraw, curBalance); err != nil {

		if err := tx.Rollback(); err != nil {
			logger.Log.Error("ошибка при откате транзакции, при изменении баланса",
				zap.String("code", err.Error()),
			)
			return fmt.Errorf("ошибка при откате транзакции, при изменении баланса: <%w>", err)
		}

		return fmt.Errorf("функция balanceWithdrawTx вернула ошибку: <%w>", err)
	}

	// Добавление списания в истоию
	if err := AddWithdrawalHistory(tx, userID, order, sumWithdraw); err != nil {

		if err := tx.Rollback(); err != nil {
			logger.Log.Error("ошибка при откате транзакции, при добавлении в историю",
				zap.String("code", err.Error()),
			)
			return fmt.Errorf("ошибка при откате транзакции, при добавлении в историю: <%w>", err)
		}

		return fmt.Errorf("функция AddWithdrawalHistory вернула ошибку: <%w>", err)
	}

	// Подтверждение транзакции
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка подтверждения транзакции: <%w>", err)
	}
	return nil
}

// Функция предоставляет историю вывода пользователя. Возвращает историю выводов и ошибку.
//
// Параметры:
//
// token - токен.
func (a *PostgresT) HistoryWithrawals(token string) ([]HistoryWithdrawalsT, error) {

	// Проверка аргументов
	if token == "" {
		return nil, errors.New("в аргументе token нет содержимого")
	}

	// Логика
	//
	// Получение ID пользователя
	userID, err := getIdByToken(a.PtrDB, token)
	if err != nil {
		return nil, fmt.Errorf("функция getIdByTokenTx вернула ошибку: <%w>", err)
	}

	// Получение истории вывода
	historyW, err := historyWithrawals(a.PtrDB, userID)
	if err != nil {
		return nil, fmt.Errorf("функция historyWithrawals, вернула ошибку: <%w>", err)
	}

	// Возврат
	return historyW, nil
}

// Функция выполняет обновление данных заказа. Возвращает ошибку.
//
// Параметры:
//
// data - данные принятые от Accrual.
func (a *PostgresT) UpdateOrder(data DataOrderAccr) error {

	// Проверка аргументов
	if data.Order == "" {
		return errors.New("недопустимое значние аргумента data.Order")
	}
	if data.Status == "" {
		return errors.New("недопустимое значние аргумента data.Status")
	}
	if data.Accrual < 0 {
		return errors.New("недопустимое значние аргумента data.Accrual")
	}

	// Логика
	//
	tx, err := a.PtrDB.Begin()
	if err != nil {
		return fmt.Errorf("ошибка создания транзакции: <%w>", err)
	}

	// Обновление статуса заказа
	userID, err := updateOrderStatusAccrual(tx, data)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("ошибка отката транзакции при обновлении статуса: <%w>", err)
		}
		return fmt.Errorf("функция updateOrderStatus, вернула ошибку: <%w>", err)
	}

	// Обновление баланса Accrual
	if err := UpdateCurrentBalance(tx, userID, data.Accrual); err != nil {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("ошибка отката транзакции при обновлении баланса: <%w>", err)
		}
		return fmt.Errorf("функция updateOrderStatus, вернула ошибку: <%w>", err)
	}

	// Удаление заказа из очереди
	if err := deleteOrderFromQueue(tx, data.Order); err != nil {
		if err := tx.Rollback(); err != nil {
			return fmt.Errorf("ошибка отката транзакции при удалении заказа из очереди: <%w>", err)
		}
		return fmt.Errorf("функция deleteOrderFromQueue, вернула ошибку: <%w>", err)
	}

	// Подтверждение транзакции
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка подтверждения транзакции: <%w>", err)
	}

	return nil
}

// Функция выполняет обновление данных заказа. Возвращает id пользователя и ошибку.
//
// Параметры:
//
// tx - транзакция.
// data - данные принятые от Accrual.
func (a *PostgresT) UpdateOrderStatus(data DataOrderAccr) error {

	// Проверка аргументов
	if data.Order == "" {
		return errors.New("недопустимое значение в data.Order")
	}
	if data.Status == "" {
		return errors.New("недопустимое значение в data.Status")
	}
	if data.Accrual < 0 {
		return errors.New("недопустимое значение в data.Accrual")
	}

	// Логика
	query := `
        UPDATE orders
        SET order_status = $1
        WHERE order_number = $2;
    `

	result, err := a.PtrDB.Exec(query, data.Status, data.Order)
	if err != nil {
		return fmt.Errorf("ошибка обновления данных заказа: <%w>", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения id затронутой строки: <%w>", err)
	}

	if rowsAffected == 0 {
		return errors.New("данные заказа не обновлены")
	}

	return nil
}

// Функция выполняет доавление необработанного заказа в очередь заказов, для последующей обработки. Возвращает ошибку.
//
// Параметры:
//
// orderNumber - номер заказа.
func (a *PostgresT) AddOrderInQueue(orderNumber string) error {

	// Проверка аргументов
	if orderNumber == "" {
		return errors.New("в аргументе orderNumber путое значение")
	}

	// Логика
	query := `INSERT INTO queue_order (order_number) VALUES ($1)`

	_, err := a.PtrDB.Exec(query, orderNumber)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении: <%w>", err)
	}

	return nil
}

// Функция получает массив заказов, ожидающих обработки. Возвращает массив заказов и ошибку.
func (a *PostgresT) GetOrdersInQueue() ([]string, error) {

	// Запрос
	query := "SELECT order_number FROM queue_order ORDER BY created_at ASC LIMIT 10"

	rows, err := a.PtrDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Log.Error("ошибка закрытия потока rows, перед выходом из функции",
				zap.String("order", err.Error()),
			)
		}
	}()

	orderNumbers := make([]string, 0)

	// Обработка запроса
	for rows.Next() {

		var orderNumber string
		if err := rows.Scan(&orderNumber); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки: <%w>", err)
		}
		orderNumbers = append(orderNumbers, orderNumber)
	}

	// Проверка на ошибки после завершения итерации
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при сканировании строк: <%w>", err)
	}

	return orderNumbers, nil
}

// Функция выполняет удалени номера заказа из очереди ожидающих. Возвращает ошибку.
//
// Параметры:
//
// orderNumber - номер заказа.
func deleteOrderFromQueue(tx *sql.Tx, orderNumber string) error {

	// Проверка аргументов
	if orderNumber == "" {
		return errors.New("нет содержимого в аргументе orderNumber")
	}

	// Логика
	query := `DELETE FROM queue_order WHERE order_number = $1`
	_, err := tx.Exec(query, orderNumber)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: <%w>", err)
	}

	return nil
}

// Функция выполняет обновление данных заказа. Возвращает id пользователя и ошибку.
//
// Параметры:
//
// tx - транзакция.
// data - данные принятые от Accrual.
func updateOrderStatusAccrual(tx *sql.Tx, data DataOrderAccr) (int64, error) {

	query := `
        UPDATE orders
        SET order_status = $1, order_accrual = $2
        WHERE order_number = $3
        RETURNING user_id;
    `

	var userID int64

	err := tx.QueryRow(query, data.Status, data.Accrual, data.Order).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("заказ не найден")
		}
		return 0, err
	}

	return userID, nil
}

// Функция выполняет обновление баланса. Возвращает ошибку.
//
// Параметры:
//
// tx - транзакция.
// userID - id пользователя.
// accrBall - принятое от Accrual значение.
func UpdateCurrentBalance(tx *sql.Tx, userID int64, accrual float64) error {

	// Проверка аргументов
	if tx == nil {
		return errors.New("в аргументе tx нет указателя")
	}
	if userID <= 0 {
		return errors.New("недопустимое значение аргумента userID")
	}
	if accrual < 0 {
		return errors.New("недопустимое значение аргумента accrBall")
	}

	// Считывание текущего балланса accrual
	querySelect := `
        SELECT accrual
        FROM balance
        WHERE user_id = $1;
    `
	var curAccrual float64

	err := tx.QueryRow(querySelect, userID).Scan(&curAccrual)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("баланс пользователя не найден")
		}
		return fmt.Errorf("ошибка считывания текущего баланса: <%w>", err)
	}

	// Добавление
	newCurrent := curAccrual + accrual

	// Обновление обновление балланса Accrual
	queryUpdate := `
        UPDATE balance
        SET accrual = $1
        WHERE user_id = $2;
    `
	_, err = tx.Exec(queryUpdate, newCurrent, userID)
	if err != nil {
		return fmt.Errorf("ошибка обновления баланса: <%w>", err)
	}

	return nil
}

// Функция формирования хеш из строки. Возвращает хеш и ошибку.
//
// Параметры:
//
// str - исходная строка.
func hashString(str string) (string, error) {

	// Проверка аргументов
	if str == "" {
		return "", errors.New("в аргументе str нет содержимого")
	}

	// Логика
	hash := sha256.New()
	hash.Write([]byte(str))
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Функция выполняет генерацию случайной строки заданной длинны. Возвращает строку и ошибку.
//
// Параметры:
//
// length - длинна строки.
func generateRandomToken(length int) (string, error) {

	// Проверка аргументов
	if length < 20 {
		return "", fmt.Errorf("принята малая длинна для генерации токена: <%d>", length)
	}

	// Логика
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(bytes)

	// Результат
	return token, nil
}

// Функция выполняет добавление пользователя в БД. Возвращает добавленный id и ошибку.
//
// Параметры:
//
// tx - транзакция.
// login - логин.
// password - пароль.
func addUserTx(tx *sql.Tx, login, password string) (int, error) {

	// Проверка аргументов
	if tx == nil {
		return 0, errors.New("в аргументе tx нет указателя")
	}
	if login == "" || password == "" {
		return 0, errors.New("в аргументе login или password, пустое значение")
	}

	// Логика
	hashPwd, err := hashString(password)
	if err != nil {
		return 0, fmt.Errorf("ошибка при создании хеша пароля: <%w>", err)
	}

	var newUserID int
	query := `INSERT INTO users (user_name, user_password) VALUES ($1, $2) RETURNING id`

	if err := tx.QueryRow(query, login, hashPwd).Scan(&newUserID); err != nil {
		return 0, fmt.Errorf("ошибка при добавлении пользователя: <%w>", err)
	}

	// Результат
	return newUserID, nil
}

// Функция выполняет чтение имени пользователя из БД по хешу пароля. Возвращает имя и ошибку.
//
// Параметры:
//
// db - указатель на БД.
// login - имя пользователя.
func getPasswordByUserNameDB(db *sql.DB, login string) (string, int64, error) {

	// Проверка аргументов
	if db == nil {
		return "", 0, errors.New("в аргументе db нет указателя")
	}
	if login == "" {
		return "", 0, errors.New("в аргументе login нет содержимого")
	}

	// Логика
	var userPwd string
	var userID int64
	query := "SELECT user_password, id FROM users WHERE user_name = $1"

	if err := db.QueryRow(query, login).Scan(&userPwd, &userID); err != nil {
		if err == sql.ErrNoRows {
			return "", 0, errors.New("пользователь не найден")
		}
		return "", 0, fmt.Errorf("ошибка запроса: <%w>", err)
	}

	// Результат
	return userPwd, userID, nil
}

// Функция выполняет обновление токена при аутентификации пользователя. Возвращает токен и ошибку.
//
// Параметры:
// db - указатель на БД.
// id - id из таблицы пользователей.
func updateTokenDB(db *sql.DB, id int64) (string, error) {

	// Проверка аргументов
	if db == nil {
		return "", errors.New("в аргументе db нет указателя")
	}
	if id < 1 {
		return "", errors.New("недопустимое значение в id")
	}

	// Логика
	var token string
	var err error

	for {
		token, err = generateRandomToken(50)
		if err != nil {
			return "", fmt.Errorf("ошибка при генерации токена: <%w>", err)
		}

		// Время действия токена - 1 час
		createdAt := time.Now()
		expiredAt := time.Now().Add(1 * time.Hour)

		// Установка поля access в true - доступ к БД (флаг авторизации)
		query := `
			INSERT INTO user_tokens (user_id, token, created_at, expired_at, access)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id) DO UPDATE
			SET token = EXCLUDED.token, created_at = EXCLUDED.created_at, expired_at = EXCLUDED.expired_at, access = EXCLUDED.access;
		`
		if _, err := db.Exec(query, id, token, createdAt, expiredAt, true); err != nil {
			if errConflictToken == err.Error() { // обнаружение конфликта токенов
				continue
			}
			return "", fmt.Errorf("ошибка обновления токена: <%w>", err)
		}
		break
	}

	// Результат
	return token, nil
}

// Функция получает id пользователя по токену. Возвращает id пользователя и ошибку.
//
// Параметры:
//
// db - указатель на БД.
// token - токен.
func getIdByToken(db *sql.DB, token string) (int64, error) {

	// Проверка аргументов
	if db == nil {
		return 0, errors.New("в аргументе db нет указателя")
	}

	// Логика
	var userID int64
	var createdAt time.Time
	var expiredAt time.Time

	query := "SELECT user_id, created_at, expired_at FROM user_tokens WHERE token = $1"

	if err := db.QueryRow(query, token).Scan(&userID, &createdAt, &expiredAt); err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("пользователь не аутентифицирован")
		}
		return 0, fmt.Errorf("ошибка запроса: <%w>", err)
	}

	// Проверка валидности времени токена
	if createdAt.Equal(expiredAt) || createdAt.After(expiredAt) {

		// Сброс авторизации
		if err := resetAccess(db, token); err != nil {
			return 0, fmt.Errorf("функция resetAccess вернула ошибку: <%w>", err)
		}

		fmt.Println("пользователь не авторизован")
	}

	// Результат
	return userID, nil
}

// Функция выполняет добавление номера заказа. Возвращает ошибку.
//
// Параметры:
//
// tx - транзакция.
// newUserID - id созданного пользователя.
func addUserOrderDB(db *sql.DB, userID int64, order string) error {

	// Проверка аргументов
	if db == nil {
		return errors.New("в аргументе tx нет указателя")
	}
	if userID < 1 {
		return errors.New("недопустимое значение userID")
	}
	if !isDigitsOnly(order) || order == "" {
		return errors.New("недопустимое значение order")
	}

	// Логика
	query := "INSERT INTO orders (user_id, order_number, order_status) VALUES ($1, $2, $3)"

	result, err := db.Exec(query, userID, order, "NEW")
	if err != nil {

		// обнаружение конфликта пары: id пользователя / номер заказа
		if errDuplicateOrderByUser == err.Error() {
			return errors.New("номер заказа уже был загружен этим пользователем")
		}
		// обнаружение конфликта номера заказа
		if errNumbOrderBusy == err.Error() {
			return errors.New("номер заказа уже был загружен другим пользователем")
		}
		return fmt.Errorf("ошибка при добавлении заказа: <%w>", err)
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return errors.New("ошибка при получении количества затронутых строк в таблице заказов")
	}
	if rowsAff != 1 {
		return errors.New("нет признака добавления записи в таблицу заказов")
	}

	// Результат
	return nil
}

// Функция выполняет проверку, что в строке только цифпы. Возвращает true или false.
//
// Параметры:
// s - проверяемая строка.
func isDigitsOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// Функция выполняет запрос к БД для получения списка заказов по id пользователя. Возвращает массив заказов и ошибку.
//
// Параметры:
//
// db - указатель на БД.
// userID - id пользователя.
func getOrdersUser(db *sql.DB, userID int64) ([]OrderT, error) {

	// Проверка аргументов
	if db == nil {
		return nil, errors.New("в аргументе tx нет указателя")
	}
	if userID < 1 {
		return nil, errors.New("отрицательное значение в аргументе userID")
	}

	// Логика
	query := `
        SELECT order_number, order_status, order_accrual, created_at
        FROM orders
        WHERE user_id = $1
    `

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]OrderT, 0)

	for rows.Next() {

		var order OrderT
		var uploadedAt time.Time

		if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &uploadedAt); err != nil {
			return nil, fmt.Errorf("ошибка при чтении содержимого строки: <%w>", err)
		}
		order.UploadedAt = uploadedAt.Format("2006-01-02T15:04:05-07:00")
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при сканировании строк ответа: <%w>", err)
	}

	// Ответ
	return orders, nil
}

// Функция выполняет сброс access токена. Возвращает ошибку.
//
// Параметры:
//
// db - указатель на БД.
// token - токен.
func resetAccess(db *sql.DB, token string) error {

	// Проверка аргументов
	if db == nil {
		return errors.New("в аргументе db нет указателя")
	}
	if token == "" {
		return errors.New("недопустимое значение токена")
	}

	// SQL-запрос для обновления поля access
	query := `
		UPDATE user_tokens
		SET access = FALSE
		WHERE token = $1;
	`

	// Выполнение запроса
	result, err := db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: <%w>", err)
	}

	// Проверка, обновлена ли запись
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка при получении количества обновленных строк: <%w>", err)
	}
	if rowsAffected == 0 {
		return errors.New("токен не найден")
	}

	return nil
}

// Функция получает баланс пользователя по его id. Возвращает балан и ошибку.
//
// Параметры:
// db - указатель на БД.
// userID - id пользователя.
func getUserBalance(db *sql.DB, userID int64) (BalanceT, error) {

	// Проверка аргументов
	if db == nil {
		return BalanceT{}, errors.New("в аргументе tx нет указателя")
	}
	if userID < 1 {
		return BalanceT{}, errors.New("недопустимое значение в userID")
	}

	// Логика
	query := `SELECT accrual, withdrawn FROM balance WHERE user_id = $1`

	row := db.QueryRow(query, userID)

	var balance BalanceT

	err := row.Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		if err == sql.ErrNoRows {
			return BalanceT{}, errors.New("данные баланса пользователя не найдены")
		}
		return BalanceT{}, fmt.Errorf("ошибка при получении баланса пользователя: <%w>", err)
	}

	// Ответ
	return balance, nil
}

// Функция выполняет списание баланса пользователя.
//
// Параметры:
//
// db - указатель на БД.
// userID - id пользователя.
// sum - сумма на списание.
// balance - текущий баланс.
func balanceWithdrawTx(tx *sql.Tx, userID int64, sum float64, balance BalanceT) error {

	// Проверка аргументов
	if tx == nil {
		return errors.New("в арнументе db нет указателя")
	}
	if userID < 1 {
		return errors.New("недопустимое содержимое userID")
	}
	if sum <= 0 {
		return errors.New("недопустимое содержимое sum")
	}
	if balance.Current < 0 {
		return errors.New("недопустимое содержимое balance.Current")
	}
	if balance.Withdrawn < 0 {
		return errors.New("недопустимое содержимое balance.Withdrawn")
	}

	// Логика
	//
	// Высисление новых показателей
	var newBalance BalanceT

	// Вычисление нового значения баланса
	currentB := balance.Current * 100.0
	currentB -= sum * 100.0
	newBalance.Current = currentB / 100.0

	// Увеличение накопителя списанных баллов
	newBalance.Withdrawn += sum

	// Обновление баланса
	if err := updateBalanceTx(tx, userID, newBalance); err != nil {
		return fmt.Errorf("функция updateBalanceTx, вернула ошибку: <%w>", err)
	}

	return nil
}

// Функция выполняет обновление балланса пользователя. Возвращает ошибку.
//
// Параметры:
//
// db - указатель на БД.
// userID - id пользователя.
// newBalance - новый баланс пользователя.
func updateBalanceTx(tx *sql.Tx, userID int64, newBalance BalanceT) error {

	// Проверка аргументов
	if tx == nil {
		return errors.New("в аргументе db нет указателя")
	}
	if userID < 1 {
		return errors.New("недопустимое содержимое в userID")
	}
	if newBalance.Current < 0 {
		return errors.New("недопустимое содержимое в newBalance.Current")
	}
	if newBalance.Withdrawn < 0 {
		return errors.New("недопустимое содержимое в newBalance.Withdrawn")
	}

	// Логика
	query := `
        UPDATE balance
        SET accrual = $1, withdrawn = $2
        WHERE user_id = $3
    `
	_, err := tx.Exec(query, newBalance.Current, newBalance.Withdrawn, userID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении баланса: %w", err)
	}
	return nil
}

// Получение истории вывода по id пользователя. Возвращается история и ошибка.
//
// Параметры:
//
// db - указатель на БД.
// userID - id пользователя.
func historyWithrawals(db *sql.DB, userID int64) ([]HistoryWithdrawalsT, error) {

	// Проверка аргументов
	if db == nil {
		return nil, errors.New("в аргументе db нет указателя")
	}
	if userID < 1 {
		return nil, errors.New("неподдерживаемый номер в userID")
	}

	// Логика
	q := `
		SELECT order_number, sum, processed_at 
        FROM withdrawals 
        WHERE user_id = $1 
        ORDER BY processed_at DESC
	`
	rows, err := db.Query(q, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: <%w>", err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			logger.Log.Error("Ошибка при выполнении rows.Close",
				zap.String("err", err.Error()),
			)
		}
	}()

	// Сбор данных
	withdrawals := make([]HistoryWithdrawalsT, 0)

	for rows.Next() {

		var withdrawal HistoryWithdrawalsT

		if err := rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.ProcessedAt); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании строки ответа: <%w>", err)
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка в сканировании строк: <%w>", err)
	}

	// Проверка результата
	if len(withdrawals) == 0 {
		return nil, errors.New("нет ни одного списания")
	}

	// Ответ
	return withdrawals, nil
}

// Функция выполняет добавление данных списания в историю. Возвращает ошибку.
//
// Параметры:
//
// tx - транзакция.
// userID - id пользователя.
// orderNumber - номер заказа.
// sum - сумма баллов на списание.
func AddWithdrawalHistory(tx *sql.Tx, userID int64, orderNumber string, sum float64) error {

	// Проверка аргументов
	if tx == nil {
		return errors.New("в аргументе tx нет указателя")
	}

	// Логика
	query := `
        INSERT INTO withdrawals (user_id, order_number, sum)
        VALUES ($1, $2, $3)
        RETURNING id;`

	result, err := tx.Exec(query, userID, orderNumber, sum)
	if err != nil {
		return fmt.Errorf("ошибка добавления записи в историю списаний: %w", err)
	}

	aff, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
	}

	if aff == 0 {
		return fmt.Errorf("запись недобавлена в историю: %w", err)
	}

	return nil
}
