package actions

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// validationByLuna

func Test_validationByLuna_SUCCESS(t *testing.T) {

	// Данные для теста
	testsData := []struct {
		nameTest   string
		number     string
		wantStatus bool
		wantErr    error
	}{
		{
			nameTest:   "Корректный номер 1",
			number:     "79927398713",
			wantStatus: true,
			wantErr:    nil,
		},
		{
			nameTest:   "Корректный номер 2",
			number:     "1234567812345670",
			wantStatus: true,
			wantErr:    nil,
		},
	}

	// Тесты
	for _, tt := range testsData {

		t.Run(tt.nameTest, func(t *testing.T) {

			result, err := isValidByLuhn(tt.number)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
			assert.Equalf(t, tt.wantStatus, result, "ожидаля <%t> а принят <%t>", tt.wantStatus, result)
		})
	}
}

func Test_validationByLuna_FAULT(t *testing.T) {

	// Данные для теста
	testsData := []struct {
		nameTest   string
		number     string
		wantStatus bool
		wantErr    error
	}{
		{
			nameTest:   "Некоректный номер",
			number:     "1234567812345678",
			wantStatus: false,
			wantErr:    nil,
		},
		{
			nameTest:   "Нет номера",
			number:     "",
			wantStatus: false,
			wantErr:    errors.New("в аргументе number нет содержимого"),
		},
		{
			nameTest:   "Неверный формат",
			number:     "12345AAA678",
			wantStatus: false,
			wantErr:    errors.New("неверный формат номера заказа"),
		},
	}

	// Тесты
	for _, tt := range testsData {

		t.Run(tt.nameTest, func(t *testing.T) {

			result, err := isValidByLuhn(tt.number)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
			assert.Equalf(t, tt.wantStatus, result, "ожидаля <%t> а принят <%t>", tt.wantStatus, result)
		})
	}
}

// IsPossibilityWithdraw

func Test_IsPossibilityWithdraw_SUCCESS(t *testing.T) {

	// Данные для теста
	testsData := []struct {
		nameTest       string
		forWithdraw    float64
		currentBalance float64
		wantResult     bool
		wantError      error
	}{
		{
			nameTest:       "Достаточно средств 1",
			forWithdraw:    10,
			currentBalance: 100,
			wantResult:     true,
			wantError:      nil,
		},
		{
			nameTest:       "Достаточно средств 2",
			forWithdraw:    10,
			currentBalance: 10,
			wantResult:     true,
			wantError:      nil,
		},
	}

	// Тесты
	for _, tt := range testsData {

		t.Run(tt.nameTest, func(t *testing.T) {

			result, err := isPossibilityWithdraw(tt.forWithdraw, tt.currentBalance)
			assert.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantError, err)
			assert.Equalf(t, tt.wantResult, result, "ожидался результат <%t> а принято <%t>", tt.wantResult, result)
		})
	}
}

func Test_IsPossibilityWithdraw_FAULT(t *testing.T) {

	// Данные для теста
	testsData := []struct {
		nameTest       string
		forWithdraw    float64
		currentBalance float64
		wantResult     bool
		wantError      error
	}{
		{
			nameTest:       "Недостаточно средств",
			forWithdraw:    100,
			currentBalance: 10,
			wantResult:     false,
			wantError:      nil,
		},
		{
			nameTest:       "Отрицательная сумма списания",
			forWithdraw:    -1,
			currentBalance: 100,
			wantResult:     false,
			wantError:      errors.New("недопустимое содержимое в аргументе forWithdraw"),
		},
		{
			nameTest:       "Ноль на списание",
			forWithdraw:    0,
			currentBalance: 100,
			wantResult:     false,
			wantError:      errors.New("недопустимое содержимое в аргументе forWithdraw"),
		},
		{
			nameTest:       "Недопустимый текущий баланс",
			forWithdraw:    1,
			currentBalance: -100,
			wantResult:     false,
			wantError:      errors.New("недопустимое содержимое в аргументе currentBalance"),
		},
		{
			nameTest:       "Недостаточно средста",
			forWithdraw:    10,
			currentBalance: 1,
			wantResult:     false,
			wantError:      nil,
		},
	}

	// Тесты
	for _, tt := range testsData {

		t.Run(tt.nameTest, func(t *testing.T) {

			result, err := isPossibilityWithdraw(tt.forWithdraw, tt.currentBalance)
			assert.Equalf(t, tt.wantError, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantError, err)
			assert.Equalf(t, tt.wantResult, result, "ожидался результат <%t> а принято <%t>", tt.wantResult, result)
		})
	}
}
