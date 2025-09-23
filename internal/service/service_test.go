package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isConnectionRefused

func Test_IsConnectionRefused(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameTest string
		errT     error
		wantT    bool
	}{
		{
			nameTest: "true-1",
			errT:     errors.New("network error: connection refused"),
			wantT:    true,
		},
		{
			nameTest: "true-2",
			errT:     errors.New("dial tcp: address 127.0.0.1: connection refused"),
			wantT:    true,
		},
		{
			nameTest: "true-3",
			errT:     errors.New("some other error: connection refused>"),
			wantT:    true,
		},
		{
			nameTest: "false-1",
			errT:     errors.New("connection reset by peer"),
			wantT:    false,
		},
		{
			nameTest: "false-2",
			errT:     errors.New("timeout error"),
			wantT:    false,
		},
		{
			nameTest: "false-3",
			errT:     errors.New("network error: connection timed out"),
			wantT:    false,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			result := isConnectionRefused(tt.errT)
			assert.Equal(t, tt.wantT, result, "ожидалось <%t> а принято <%t>", tt.wantT, result)

		})
	}
}

// isResponceTimeout

func Test_IsResponceTimeout(t *testing.T) {

	// Данные
	testData := []struct {
		nameTest string
		errT     error
		wantT    bool
	}{
		{
			nameTest: "true-1",
			errT:     fmt.Errorf("network error: <%w>", errors.New(ErrTimeoutResponceAccrual)),
			wantT:    true},
		{
			nameTest: "true-2",
			errT:     fmt.Errorf("some other error: <%w>", errors.New(ErrTimeoutResponceAccrual)),
			wantT:    true,
		},
		{
			nameTest: "true-3",
			errT:     fmt.Errorf("timeout error: <%w>", errors.New(ErrTimeoutResponceAccrual+">")),
			wantT:    true,
		},
		{
			nameTest: "false",
			errT:     fmt.Errorf("connection reset by peer: <%w>", errors.New("aaa")),
			wantT:    false,
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {

			result := isResponceTimeout(tt.errT)
			assert.Equalf(t, tt.wantT, result, "ожидалось <%t> а принято <%t>", tt.wantT, result)

		})
	}
}

// checkNetErrors

func Test_checkNetErrors(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameTest string
		errT     error
		wantT    bool
	}{
		{
			nameTest: "refused",
			errT:     fmt.Errorf("network error: <%w>", errors.New("dial tcp: address 127.0.0.1: connection refused")),
			wantT:    true,
		},
		{
			nameTest: "timeout",
			errT:     fmt.Errorf("network error: <%w>", errors.New("Client.Timeout exceeded while awaiting headers")),
			wantT:    true,
		},
		{
			nameTest: "false - 1",
			errT:     fmt.Errorf("network error: <%w>", errors.New("dial tcp: address 127.0.0.1: aaa")),
			wantT:    false,
		},
		{
			nameTest: "false - 2",
			errT:     fmt.Errorf("network error: <%w>", errors.New("Client.Timeout")),
			wantT:    false,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			result, err := checkNetErrors(tt.errT)
			require.NoErrorf(t, err, "неожиданная ошибка <%v>", err)
			assert.Equal(t, tt.wantT, result, "ожидалось <%t> а принято <%t>", tt.wantT, result)

		})
	}
}

// checkOtherErrors

func Test_CheckOtherErrors(t *testing.T) {
	// Данные для тестов
	testsData := []struct {
		nameTest              string
		errT                  error
		wantNeedStoreByErr    bool // флаг - ошибки сервера
		wantNeedStoreByFreq   bool // флаг - 429
		wantNeedStoreByNotReg bool // флаг - заказ незарегистрирован
		wantDo                bool
	}{
		{
			nameTest:              "ErrorNotExistOrder",
			errT:                  errors.New(ErrAccrNotExistOrder),
			wantNeedStoreByErr:    false,
			wantNeedStoreByFreq:   false,
			wantNeedStoreByNotReg: false,
			wantDo:                true,
		},

		{
			nameTest:              "ErrorTooManyRequests",
			errT:                  errors.New(ErrAccrTooManyRequests),
			wantNeedStoreByErr:    false,
			wantNeedStoreByFreq:   true,
			wantNeedStoreByNotReg: false,
			wantDo:                true,
		},
		{
			nameTest:              "ErrorInternalServerErr",
			errT:                  errors.New(ErrAccrInternalServerErr),
			wantNeedStoreByErr:    false,
			wantNeedStoreByFreq:   false,
			wantNeedStoreByNotReg: false,
			wantDo:                true,
		},
		{
			nameTest: "Неизвестная ошибка",
			errT:     errors.New("unknown error"),
			wantDo:   false,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameTest, func(t *testing.T) {

			params := &ProcessingOrderT{
				tmr:                   &time.Timer{},
				newOrdNumb:            "12345",
				needStoreByFreq:       false,
				needStoreByErr:        false,
				needStoreByNotReg:     false,
				needStoreByREGISTERED: false,
				needStoreByPROCESSING: false,
				adptAccr:              nil,
				adptPG:                nil,
			}

			result, _ := checkOtherErrors(tt.errT, params)
			assert.Equalf(t, tt.wantDo, result, "ожидалось <%t> а принято <%t>", tt.wantDo, result)
			assert.Equalf(t, tt.wantNeedStoreByErr, params.needStoreByErr, " wantNeedStoreByErr ожидалось <%t> а принято <%t>", tt.wantNeedStoreByErr, params.needStoreByErr)
			assert.Equalf(t, tt.wantNeedStoreByNotReg, params.needStoreByNotReg, " needStoreByNotReg ожидалось <%t> а принято <%t>", tt.wantNeedStoreByNotReg, params.needStoreByNotReg)
			assert.Equalf(t, tt.wantNeedStoreByFreq, params.needStoreByFreq, " needStoreByFreq ожидалось <%t> а принято <%t>", tt.wantNeedStoreByFreq, params.needStoreByFreq)
		})
	}
}
