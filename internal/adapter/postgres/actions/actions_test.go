package actionspg

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RegisterUser

func Test_RegisterUser_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testData := []struct {
		nameTest  string
		login     string
		password  string
		initMockT func(mock sqlmock.Sqlmock)
		wantID    int
	}{
		{
			nameTest: "Корректные данные",
			login:    "AAA",
			password: "BBB",
			initMockT: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				hashPwd, _ := hashString("BBB")
				mock.ExpectQuery("INSERT INTO users").WithArgs("AAA", hashPwd).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
			wantID: 1,
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.initMockT(mock)

			id, err := adptPG.RegisterUser(tt.login, tt.password)
			require.NoError(t, err, "ошибка: <%v>", err)
			assert.Equal(t, tt.wantID, id, "ожидался id <%d>, а принят <%d>", tt.wantID, id)

			if err := mock.ExpectationsWereMet(); err != nil {
				assert.NoErrorf(t, err, "не все ожидания были выполнены: <%v>", err)
			}
		})
	}
}

func Test_RegisterUser_FAULT(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testData := []struct {
		nameTest  string
		login     string
		password  string
		initMockT func(mock sqlmock.Sqlmock)
		wantErr   string
	}{
		{
			nameTest: "Нет данных Login",
			login:    "",
			password: "BBB",
			initMockT: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				hashPwd, _ := hashString("BBB")
				mock.ExpectQuery("INSERT INTO users").WithArgs("", hashPwd).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
			wantErr: "в аргументе login нет содержимого",
		},
		{
			nameTest: "Нет данных Password",
			login:    "AAA",
			password: "",
			initMockT: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				hashPwd, _ := hashString("")
				mock.ExpectQuery("INSERT INTO users").WithArgs("AAA", hashPwd).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectCommit()
			},
			wantErr: "в аргументе password нет содержимого",
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.initMockT(mock)

			_, err := adptPG.RegisterUser(tt.login, tt.password)
			assert.Equal(t, tt.wantErr, err.Error())
		})
	}
}

// CreateUserBalance

func Test_CreateUserBalance_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testData := []struct {
		nameTest  string
		userID    int
		initMockT func(mock sqlmock.Sqlmock)
	}{
		{
			nameTest: "Корректные данные",
			userID:   1,
			initMockT: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO balance").WithArgs(1, 0, 0).WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {
			tt.initMockT(mock)

			err := adptPG.CreateUserBalance(tt.userID)
			require.NoError(t, err, "ошибка: <%v>", err)

			if err := mock.ExpectationsWereMet(); err != nil {
				assert.NoErrorf(t, err, "не все ожидания были выполнены: <%v>", err)
			}
		})
	}
}

func Test_CreateUserBalance_FAULT(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testData := []struct {
		nameTest  string
		userID    int
		initMockT func(mock sqlmock.Sqlmock)
		wantErr   string
	}{
		{
			nameTest: "Недопустимый userID",
			userID:   0,
			initMockT: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO balance").WithArgs(1, 0, 0).WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: "недопустимое содержимое в аргумете userID",
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {
			tt.initMockT(mock)

			err := adptPG.CreateUserBalance(tt.userID)
			assert.Equalf(t, tt.wantErr, err.Error(), "ожадалась ошибка: <%s>, а принято <%s>", tt.wantErr, err.Error())
		})
	}
}

// AuthenticationUser

func Test_GetPasswordByUserNameDB_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoErrorf(t, err, "ошибка создания mock: <%v>", err)

	defer db.Close()

	// Данные для теста
	testsData := []struct {
		nameTest  string
		login     string
		mockSetup func()
		wantPwd   string
		wantID    int64
	}{
		{
			nameTest: "Пользователь найден",
			login:    "testuser",
			mockSetup: func() {
				mock.ExpectQuery("SELECT user_password, id FROM users").
					WithArgs("testuser").
					WillReturnRows(sqlmock.NewRows([]string{"user_password", "id"}).AddRow("AAA", 1))
			},
			wantPwd: "AAA",
			wantID:  1,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mockSetup()

			pwd, id, err := getPasswordByUserNameDB(db, tt.login)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.wantPwd, pwd, "ожидался пароль <%s>, а принят <%s>", tt.wantPwd, pwd)
			assert.Equalf(t, tt.wantID, id, "ожидался id <%s>, а принят <%s>", tt.wantID, id)
		})
	}
}

func Test_GetPasswordByUserNameDB_FAULT(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Не удалось создать мок базы данных: %s", err)
	}
	defer db.Close()

	// Данные для теста
	tests := []struct {
		name      string
		login     string
		mockSetup func()
		wantPwd   string
		wantID    int64
		wantError error
	}{
		{
			name:  "Пользователь не найден",
			login: "unknownuser",
			mockSetup: func() {
				mock.ExpectQuery("SELECT user_password, id FROM users").
					WithArgs("unknownuser").
					WillReturnError(sql.ErrNoRows)
			},
			wantPwd:   "",
			wantID:    0,
			wantError: errors.New("пользователь не найден"),
		},
		{
			name:      "Пустой login",
			login:     "",
			mockSetup: func() {},
			wantPwd:   "",
			wantID:    0,
			wantError: errors.New("в аргументе login нет содержимого"),
		},
		{
			name:      "Нет указателя db",
			login:     "",
			mockSetup: func() {},
			wantPwd:   "",
			wantID:    0,
			wantError: errors.New("в аргументе db нет указателя"),
		},
	}

	// Тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.mockSetup()

			if tt.name == "Нет указателя db" {
				db = nil
			}

			_, _, err = getPasswordByUserNameDB(db, tt.login)
			assert.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v>, а принято <%v>", tt.wantError, err)

		})
	}
}

func Test_updateTokenDB_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoErrorf(t, err, "ошибка создания mock: <%v>", err)

	defer db.Close()

	// Данные для теста
	tests := []struct {
		name       string
		userID     int64
		mockExpect func()
	}{
		{
			name:   "Успешное обновление токена",
			userID: 1,
			mockExpect: func() {
				mock.ExpectExec("INSERT INTO user_tokens").
					WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), true).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	// Тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.mockExpect()

			token, err := updateTokenDB(db, tt.userID)
			require.NoErrorf(t, err, "неожиланная ошибка: <%v>", err)
			assert.NotEmptyf(t, token, "в токене нет содержимого")
		})
	}
}

func Test_updateTokenDB_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoErrorf(t, err, "ошибка создания mock: <%v>", err)

	defer db.Close()

	// Данные для теста
	testsData := []struct {
		name      string
		id        int64
		wantError error
	}{
		{
			name:      "Недопустимое значение id",
			id:        0,
			wantError: errors.New("недопустимое значение в id"),
		},
		{
			name:      "Нет указателя на db",
			id:        1,
			wantError: errors.New("в аргументе db нет указателя"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.name, func(t *testing.T) {

			if tt.name == "Нет указателя на db" {
				db = nil
			}

			_, err = updateTokenDB(db, tt.id)
			assert.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принятом <%v>", tt.wantError, err)

		})
	}
}

// AddOrder

func Test_AddOrder_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest string
		order    string
		userID   int64
		mock     func()
	}{
		{
			nameTest: "Успешное добавление заказа",
			order:    "12345",
			userID:   1,
			mock: func() {
				mock.ExpectExec("INSERT INTO orders").
					WithArgs(1, "12345", "NEW").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mock()

			err := adptPG.AddOrder(tt.order, tt.userID)
			require.NoErrorf(t, err, "ошибка: <%v>", err)
		})
	}
}

func Test_AddOrder_FAULT(t *testing.T) {
	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	tests := []struct {
		nameTest string
		order    string
		userID   int64
		mock     func()
		wantErr  error
	}{

		{
			nameTest: "Нет номера заказа",
			order:    "",
			userID:   1,
			mock:     func() {},
			wantErr:  errors.New("в аргументе order нет содержимого"),
		},
		{
			nameTest: "Недопустимый userID",
			order:    "12345",
			userID:   0,
			mock:     func() {},
			wantErr:  errors.New("в аргументе userID недопустимое значение"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mock()

			err := adptPG.AddOrder(tt.order, tt.userID)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)

		})
	}
}

// GetOrdersUser

func Test_GetOrdersUser_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста

	tn := time.Now()

	testsData := []struct {
		nameTest     string
		token        string
		orderNumb    string
		orderStatus  string
		orderAccrual float64
		mock         func()
	}{
		{
			nameTest:     "Успешное чтение",
			token:        "12345",
			orderNumb:    "123",
			orderStatus:  "PROCESSED",
			orderAccrual: 100.0,
			mock: func() {

				// userID по токену
				mock.ExpectQuery("SELECT user_id, created_at, expired_at FROM user_tokens").
					WithArgs("12345").
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "created_at", "expired_at"}).
						AddRow(1, tn, tn.Add(1*time.Hour)))

				// Заказы пользователя
				mock.ExpectQuery("SELECT order_number, order_status, order_accrual, created_at FROM orders WHERE user_id = \\$1").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"order_number", "order_status", "order_accrual", "created_at"}).
						AddRow("123", "PROCESSED", 100.0, tn))
			},
		},
	}

	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mock()

			result, err := adptPG.GetOrdersUser(tt.token)
			require.NoErrorf(t, err, "ошибка: <%v>", err)
			assert.Equalf(t, tt.orderNumb, result[0].Number, "ожидался номер <%s> а принят <%s>", tt.orderNumb, result[0].Number)
			assert.Equalf(t, tt.orderStatus, result[0].Status, "ожидался статус <%s> а принят <%s>", tt.orderStatus, result[0].Status)
			assert.Equalf(t, tt.orderAccrual, result[0].Accrual, "ожидались баллы <%s> а принято <%s>", tt.orderAccrual, result[0].Accrual)

			strTN := tn.Format("2006-01-02T15:04:05-07:00")
			assert.Equalf(t, strTN, result[0].UploadedAt, "ожидалось время <%s> а принято <%s>", strTN, result[0].UploadedAt)

			if err := mock.ExpectationsWereMet(); err != nil {
				assert.NoErrorf(t, err, "не все ожидания были выполнены: <%v>", err)
			}
		})
	}
}

func Test_GetOrdersUser_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest string
		token    string
		wantErr  error
	}{
		{
			nameTest: "нет токена",
			token:    "",
			wantErr:  errors.New("в аргументе token нет содержимого"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			_, err = adptPG.GetOrdersUser(tt.token)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// GetUserBalance

func Test_GetUserBalance_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		testName    string
		userID      int64
		mock        func()
		wantBalance BalanceT
		wantErr     error
	}{
		{
			testName: "данные есть",
			userID:   1,
			mock: func() {
				mock.ExpectQuery(`SELECT accrual, withdrawn FROM balance`).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"accrual", "withdrawn"}).AddRow(100, 50))
			},
			wantBalance: BalanceT{Current: 100.0, Withdrawn: 50.0},
			wantErr:     nil,
		},
		{
			testName: "данных нет",
			userID:   2,
			mock: func() {
				mock.ExpectQuery(`SELECT accrual, withdrawn FROM balance`).
					WithArgs(2).
					WillReturnError(sql.ErrNoRows)
			},
			wantBalance: BalanceT{},
			wantErr:     errors.New("данные баланса пользователя не найдены"),
		},
	}

	for _, tt := range testsData {
		t.Run(tt.testName, func(t *testing.T) {

			tt.mock()

			balance, err := adptPG.GetUserBalance(tt.userID)
			if tt.testName == "данные есть" {
				assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
				assert.Equalf(t, tt.wantBalance.Current, balance.Current, "ожидался балланс <%v> а принято <%v>", tt.wantBalance.Current, balance.Current)
				assert.Equalf(t, tt.wantBalance.Withdrawn, balance.Withdrawn, "ожидалось списание <%v> а принято <%v>", tt.wantBalance.Withdrawn, balance.Withdrawn)
			}
			if tt.testName == "данных нет" {
				er := errors.Unwrap(err)
				assert.Equalf(t, tt.wantErr.Error(), er.Error(), "ожидалась ошибка <%v> а принято <%v>", tt.wantErr.Error(), er.Error())
				assert.Equalf(t, tt.wantBalance.Current, balance.Current, "ожидался балланс <%v> а принято <%v>", tt.wantBalance.Current, balance.Current)
				assert.Equalf(t, tt.wantBalance.Withdrawn, balance.Withdrawn, "ожидалось списание <%v> а принято <%v>", tt.wantBalance.Withdrawn, balance.Withdrawn)
			}

			// Проверяем, что все ожидания были выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Не все ожидания были выполнены: %s", err)
			}
		})
	}

	// Проверяем, что все ожидания были выполнены
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Не все ожидания были выполнены: %s", err)
	}
}

func Test_GetUserBalance_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		testName string
		userID   int64
		mock     func()
		wantErr  error
	}{
		{
			testName: "недопустимое значение userID",
			userID:   0,
			mock:     func() {},
			wantErr:  errors.New("недопустимое значение в userID"),
		},
	}

	for _, tt := range testsData {
		t.Run(tt.testName, func(t *testing.T) {

			_, err := adptPG.GetUserBalance(tt.userID)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// GetUserIDByToken

func Test_GetUserIDByToken_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		name       string
		token      string
		mockSetup  func()
		wantUserID int64
		wantError  error
	}{
		{
			name:  "Успешное получение userID",
			token: "valid_token",
			mockSetup: func() {
				mock.ExpectQuery("SELECT user_id, created_at, expired_at FROM user_tokens WHERE token = \\$1").
					WithArgs("valid_token").
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "created_at", "expired_at"}).
						AddRow(1, time.Now(), time.Now().Add(1*time.Hour)))
			},
			wantUserID: 1,
			wantError:  nil,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.name, func(t *testing.T) {

			tt.mockSetup()

			userID, err := adptPG.GetUserIDByToken(tt.token)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.wantUserID, userID, "ожидался ID <%d> а принят <%d>", tt.wantUserID, userID)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Не все ожидания были выполнены: %s", err)
			}
		})
	}
}

func Test_GetUserIDByToken_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		name      string
		token     string
		wantError error
	}{
		{
			name:      "нет токена",
			token:     "",
			wantError: errors.New("в аргументе token нет содержимого"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.name, func(t *testing.T) {

			_, err := adptPG.GetUserIDByToken(tt.token)
			require.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantError, err)
		})
	}
}

// DoWithdraw

func Test_DoWithdraw_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		name          string
		userID        int64
		sumWithdraw   float64
		curBalance    BalanceT
		order         string
		mockSetup     func()
		expectedError error
	}{
		{
			name:        "Успешное списание",
			userID:      1,
			sumWithdraw: 10.0,
			curBalance:  BalanceT{Current: 200.0, Withdrawn: 0.0},
			order:       "order123",
			mockSetup: func() {
				mock.ExpectBegin()

				mock.ExpectExec("UPDATE balance").
					WithArgs(190.0, 10.0, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec("INSERT INTO withdrawals").
					WithArgs(1, "order123", 10.0).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectCommit()
			},
			expectedError: nil,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err = adptPG.DoWithdraw(tt.userID, tt.sumWithdraw, tt.curBalance, tt.order)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)

			// Проверка, что все ожидания были выполнены
			err = mock.ExpectationsWereMet()
			require.NoError(t, err, "не все ожидания были выполнены")
		})
	}
}

func Test_DoWithdraw_FAULT(t *testing.T) {
	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		name        string
		userID      int64
		sumWithdraw float64
		curBalance  BalanceT
		order       string
		wantError   error
	}{
		{
			name:        "недопустимый userID",
			userID:      0,
			sumWithdraw: 10,
			curBalance: BalanceT{
				Current:   100,
				Withdrawn: 0,
			},
			order:     "AAA",
			wantError: errors.New("в аргументе userID недопустимое зночение"),
		},
		{
			name:        "недопустимое зночение списания",
			userID:      1,
			sumWithdraw: 0,
			curBalance: BalanceT{
				Current:   100,
				Withdrawn: 0,
			},
			order:     "AAA",
			wantError: errors.New("в аргументе sumWithdraw недопустимое зночение"),
		},
		{
			name:        "недопустимое зночение баланса",
			userID:      1,
			sumWithdraw: 1,
			curBalance: BalanceT{
				Current:   0,
				Withdrawn: 10,
			},
			order:     "AAA",
			wantError: errors.New("в аргументе curBalance.Current недопустимое зночение"),
		},
		{
			name:        "недопустимое зночение заказа",
			userID:      1,
			sumWithdraw: 1,
			curBalance: BalanceT{
				Current:   10,
				Withdrawn: 1,
			},
			order:     "",
			wantError: errors.New("в аргументе order недопустимое зночение"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.name, func(t *testing.T) {

			err = adptPG.DoWithdraw(tt.userID, tt.sumWithdraw, tt.curBalance, tt.order)
			require.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantError, err)
		})
	}
}

// HistoryWithrawals

func Test_HistoryWithrawals_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)
	tn := time.Now()

	// Данные для теста
	testsData := []struct {
		nameTest  string
		token     string
		mockSetup func()
		wantHist  []HistoryWithdrawalsT
		wantErr   error
	}{
		{
			nameTest: "Успешный случай",
			token:    "valid_token",
			mockSetup: func() {

				mock.ExpectQuery("SELECT user_id, created_at, expired_at FROM user_tokens").
					WithArgs("valid_token").
					WillReturnRows(sqlmock.NewRows([]string{"user_id", "created_at", "expired_at"}).
						AddRow(1, tn, tn.Add(1*time.Hour)))

				mock.ExpectQuery("SELECT order_number, sum, processed_at FROM withdrawals").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"order_number", "sum", "processed_at"}).
						AddRow("order1", 100.0, tn))
			},
			wantHist: []HistoryWithdrawalsT{{Order: "order1", Sum: 100.0, ProcessedAt: tn}},
			wantErr:  nil,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mockSetup()

			result, err := adptPG.HistoryWithrawals(tt.token)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.wantHist[0].Order, result[0].Order, "ожидался заказ <%s> а принят <%s>", tt.wantHist[0].Order, result[0].Order)
			assert.Equalf(t, tt.wantHist[0].Sum, result[0].Sum, "ожидалась сумма <%s> а принято <%s>", tt.wantHist[0].Sum, result[0].Sum)
			assert.Equalf(t, tn, result[0].ProcessedAt, "ожидалось время <%s> а принято <%s>", tn, result[0].ProcessedAt)

			// Проверяем, что все ожидания mock выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Не все ожидания выполнены: %s", err)
			}
		})
	}
}

func Test_HistoryWithrawals_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest string
		token    string
		wantErr  error
	}{
		{
			nameTest: "недопустимое значение token",
			token:    "",
			wantErr:  errors.New("в аргументе token нет содержимого"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			_, err = adptPG.HistoryWithrawals(tt.token)
			require.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// UpdateOrder

func Test_UpdateOrder_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest  string
		data      DataOrderAccr
		mock      func()
		wantError error
	}{
		{
			nameTest: "Успешное обновление заказа",
			data: DataOrderAccr{
				Order:   "12345",
				Status:  "PROCESSED",
				Accrual: 100.0,
			},
			mock: func() {
				mock.ExpectBegin()

				mock.ExpectQuery(`UPDATE orders SET order_status`).
					WithArgs("PROCESSED", 100.0, "12345").
					WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(1))

				mock.ExpectQuery(`SELECT accrual FROM balance`).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"accrual"}).AddRow(50.0))

				mock.ExpectExec(`UPDATE balance SET accrual`).
					WithArgs(150.0, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectExec(`DELETE FROM queue_order WHERE order_number`).
					WithArgs("12345").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectCommit()
			},
			wantError: nil,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mock()

			err := adptPG.UpdateOrder(tt.data)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)

			// Проверяем, что все ожидания mock выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Не все ожидания выполнены: %s", err)
			}
		})
	}
}

func Test_UpdateOrder_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest  string
		data      DataOrderAccr
		wantError error
	}{
		{
			nameTest: "пустой data.Order",
			data: DataOrderAccr{
				Order:   "",
				Status:  "PROCESSED",
				Accrual: 100.0,
			},
			wantError: errors.New("недопустимое значние аргумента data.Order"),
		},
		{
			nameTest: "пустой data.Status",
			data: DataOrderAccr{
				Order:   "123",
				Status:  "",
				Accrual: 100.0,
			},
			wantError: errors.New("недопустимое значние аргумента data.Status"),
		},
		{
			nameTest: "пустой data.Accrual",
			data: DataOrderAccr{
				Order:   "123",
				Status:  "PROCESSED",
				Accrual: -1,
			},
			wantError: errors.New("недопустимое значние аргумента data.Accrual"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			err := adptPG.UpdateOrder(tt.data)
			require.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantError, err)
		})
	}
}

// UpdateOrderStatus

func Test_UpdateOrderStatus_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest  string
		data      DataOrderAccr
		mock      func()
		wantError error
	}{
		{
			nameTest: "Успешное обновление статуса заказа",
			data:     DataOrderAccr{Order: "123", Status: "PROCESSED"},
			mock: func() {
				mock.ExpectExec("UPDATE orders").
					WithArgs("PROCESSED", "123").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: nil,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mock()

			err := adptPG.UpdateOrderStatus(tt.data)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)

			// Проверяем, что все ожидания были выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Не все ожидания были выполнены: %s", err)
			}
		})
	}
}

func Test_UpdateOrderStatus_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest  string
		data      DataOrderAccr
		wantError error
	}{
		{
			nameTest:  "значение data.Order",
			data:      DataOrderAccr{Order: "", Status: "PROCESSED", Accrual: 100.0},
			wantError: errors.New("недопустимое значение в data.Order"),
		},
		{
			nameTest:  "значение data.Status",
			data:      DataOrderAccr{Order: "123", Status: "", Accrual: 100.0},
			wantError: errors.New("недопустимое значение в data.Status"),
		},
		{
			nameTest:  "значение data.Accrual",
			data:      DataOrderAccr{Order: "123", Status: "PROCESSED", Accrual: -1},
			wantError: errors.New("недопустимое значение в data.Accrual"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			err := adptPG.UpdateOrderStatus(tt.data)
			require.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantError, err)
		})
	}
}

// AddOrderInQueue

func Test_AddOrderInQueue_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest    string
		orderNumber string
		mock        func()
		wantError   error
	}{
		{
			nameTest:    "Успешное добавление заказа",
			orderNumber: "12345",
			mock: func() {
				mock.ExpectExec(`INSERT INTO queue_order`).
					WithArgs("12345").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantError: nil,
		},
	}

	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mock()

			err := adptPG.AddOrderInQueue(tt.orderNumber)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)

			// Проверяем, что все ожидания были выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Не все ожидания были выполнены: %s", err)
			}
		})
	}
}

func Test_AddOrderInQueue_FAULT(t *testing.T) {

	// Подготовка
	db, _, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest    string
		orderNumber string
		wantError   error
	}{
		{
			nameTest:    "нет содержимого orderNumber",
			orderNumber: "",
			wantError:   errors.New("в аргументе orderNumber путое значение"),
		},
	}

	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			err := adptPG.AddOrderInQueue(tt.orderNumber)
			require.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantError, err)
		})
	}
}

// GetOrdersInQueue

func Test_GetOrdersInQueue_SUCCESS(t *testing.T) {

	// Подготовка
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "ошибка при создании sqlmock: <%v>", err)

	defer func() {
		_ = db.Close()
	}()

	adptPG := NewInstAdapterPostgres(db)

	// Данные для теста
	testsData := []struct {
		nameTest   string
		mock       func()
		wantOrders []string
		wantError  error
	}{
		{
			nameTest: "Успешное получение заказов",
			mock: func() {

				rows := sqlmock.NewRows([]string{"order_number"}).
					AddRow("order1").
					AddRow("order2").
					AddRow("order3")

				mock.ExpectQuery("SELECT order_number FROM queue_order ORDER BY created_at ASC").
					WillReturnRows(rows)
			},
			wantOrders: []string{"order1", "order2", "order3"},
			wantError:  nil,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			tt.mock()

			_, err := adptPG.GetOrdersInQueue()
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)

			// Проверяем, что все ожидания были выполнены
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Не все ожидания были выполнены: %s", err)
			}
		})
	}
}
