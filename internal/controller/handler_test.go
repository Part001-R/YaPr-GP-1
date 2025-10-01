package controller

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Part001-R/YaPr-GP-1/internal/service/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Middleware

func Test_EncodingMiddleware_SUCCESS(t *testing.T) {

	// Обработчик для теста Middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mw := encodingMiddleware(testHandler)

	// Данные для теста
	testData := []struct {
		nameTest   string
		methodReqT string
		reqURLT    string
		encodingT  string
		wantCodeT  int
	}{
		{
			nameTest:   "успешный запрос с gzip",
			methodReqT: http.MethodGet,
			reqURLT:    "/api/user/test",
			encodingT:  "gzip",
			wantCodeT:  http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {

			req := httptest.NewRequest(tt.methodReqT, tt.reqURLT, nil)
			req.Header.Set("Accept-Encoding", tt.encodingT)
			req.Header.Set("Authorization", "aaa")

			res := httptest.NewRecorder()

			mw.ServeHTTP(res, req)
			require.Equalf(t, tt.wantCodeT, res.Code, "ожидали код %d, получили %d", tt.wantCodeT, res.Code)

			reader, err := gzip.NewReader(res.Body)
			require.NoErrorf(t, err, "неожиданная ошибка NewReader <%v>", err)
			defer func() {
				_ = reader.Close()
			}()

			decompressedBody, err := io.ReadAll(reader)
			require.NoErrorf(t, err, "неожиданная ошибка чтения тела <%v>", err)
			defer func() {
				_ = reader.Close()
			}()

			assert.Equal(t, "OK", string(decompressedBody))
		})
	}
}

func Test_EncodingMiddleware_ERROR(t *testing.T) {

	// Обработчик для теста Middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mw := encodingMiddleware(testHandler)

	// Данные для теста
	testData := []struct {
		nameTest   string
		methodReqT string
		reqURLT    string
		encodingT  string
		wantCodeT  int
	}{
		{
			nameTest:   "неподдерживаемая кодировка Accept-Encoding",
			methodReqT: http.MethodGet,
			reqURLT:    "/api/user/test",
			encodingT:  "unsupported-encoding",
			wantCodeT:  http.StatusBadRequest,
		},
		{
			nameTest:   "неправильная кодировка Content-Encoding",
			methodReqT: http.MethodPost,
			reqURLT:    "/api/user/test",
			encodingT:  "gzippp",
			wantCodeT:  http.StatusBadRequest,
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {

			var reqBody io.Reader
			if tt.methodReqT == http.MethodPost {
				reqBody = strings.NewReader("bbb")
			}

			req := httptest.NewRequest(tt.methodReqT, tt.reqURLT, reqBody)
			req.Header.Set("Accept-Encoding", tt.encodingT)

			if tt.wantCodeT == http.StatusUnauthorized {

			} else {
				req.Header.Set("Authorization", "aaa")
			}

			res := httptest.NewRecorder()

			mw.ServeHTTP(res, req)

			require.Equalf(t, tt.wantCodeT, res.Code, "ожидали код %d, получили %d", tt.wantCodeT, res.Code)
		})
	}
}

func Test_AuthorizationMiddleware_SUCCESS(t *testing.T) {

	// Обработчик для теста Middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authorized"))
	})

	mw := authorizationMiddleware(testHandler)

	// Данные для теста
	testData := []struct {
		nameTest  string
		methodReq string
		reqURL    string
		token     string
		wantCode  int
	}{
		{
			nameTest:  "успешный запрос с Authorization",
			methodReq: http.MethodGet,
			reqURL:    "/api/user/test",
			token:     "AAA",
			wantCode:  http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {

			req := httptest.NewRequest(tt.methodReq, tt.reqURL, nil)
			req.Header.Set("Authorization", tt.token)

			res := httptest.NewRecorder()

			mw.ServeHTTP(res, req)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.Equalf(t, tt.wantCode, res.Code, "ожидали код %d, получили %d", tt.wantCode, res.Code)
		})
	}
}

func Test_AuthorizationMiddleware_FAULT(t *testing.T) {

	// Обработчик для теста Middleware
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authorized"))
	})

	mw := authorizationMiddleware(testHandler)

	// Данные для теста
	testData := []struct {
		nameTest  string
		methodReq string
		reqURL    string
		wantCode  int
	}{
		{
			nameTest:  "Без Authorization",
			methodReq: http.MethodGet,
			reqURL:    "/api/user/test",
			wantCode:  http.StatusUnauthorized,
		},
	}

	// Тесты
	for _, tt := range testData {
		t.Run(tt.nameTest, func(t *testing.T) {

			req := httptest.NewRequest(tt.methodReq, tt.reqURL, nil)

			res := httptest.NewRecorder()

			mw.ServeHTTP(res, req)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.Equalf(t, tt.wantCode, res.Code, "ожидали код %d, получили %d", tt.wantCode, res.Code)
		})
	}
}

// Register

func Test_UserRegisterLayerRx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT        string
		urlT         string
		bodyT        RegisterRx
		contentTypeT string
	}{
		{
			nameT: "Корректные данные",
			urlT:  "/aaa",
			bodyT: RegisterRx{
				Login:    "AAA",
				Password: "BBB",
			},
			contentTypeT: "application/json",
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			bodyBytes, err := json.Marshal(tt.bodyT)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			req := httptest.NewRequest(http.MethodPost, tt.urlT, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", tt.contentTypeT)

			result, err := UserRegisterLayerRx(req)
			require.NoErrorf(t, err, "неожиданная ошибка UserRegisterLayerRx: <%v>", err)
			assert.Equalf(t, tt.bodyT.Login, result.Login, "ожидался логин <%s>, а принято <%s>", tt.bodyT.Login, result.Login)
			assert.Equalf(t, tt.bodyT.Password, result.Password, "ожидался пароль <%s>, а принято <%s>", tt.bodyT.Password, result.Password)
		})
	}
}

func Test_UserRegisterLayerRx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT       string
		urlT        string
		body        RegisterRx
		contentType string
		wantErr     error
	}{
		{
			nameT: "пустой Login",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "",
				Password: "BBB",
			},
			contentType: "application/json",
			wantErr:     errors.New("400"),
		},
		{
			nameT: "пустой Password",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "AAA",
				Password: "",
			},
			contentType: "application/json",
			wantErr:     errors.New("400"),
		},
		{
			nameT: "Content-Type",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "AAA",
				Password: "BBB",
			},
			contentType: "aaa",
			wantErr:     errors.New("400"),
		},
		{
			nameT: "req == nil",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "AAA",
				Password: "BBB",
			},
			contentType: "application/json",
			wantErr:     errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			bodyBytes, err := json.Marshal(tt.body)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			req := httptest.NewRequest(http.MethodPost, tt.urlT, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", tt.contentType)

			if tt.nameT == "req == nil" {
				req = nil
			}

			_, err = UserRegisterLayerRx(req)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

func Test_UserRegisterLayerTx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT      string
		tokenT     string
		wantErr    error
		wantStatus int
	}{
		{
			nameT:      "Корректные данные",
			tokenT:     "token",
			wantErr:    nil,
			wantStatus: http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			err := UserRegisterLayerTx(res, tt.tokenT)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.Equalf(t, tt.wantErr, err, "ожидалось <%v> а принято <%v>", tt.wantErr, err)
			assert.Equalf(t, tt.wantStatus, resp.StatusCode, "ожидался код <%d> а принято <%d>", tt.wantStatus, resp.StatusCode)
			assert.Equalf(t, tt.tokenT, resp.Header.Get("Authorization"), "ожидался токена <%s> а принято <%s>", tt.tokenT, resp.Header.Get)

		})
	}
}

func Test_UserRegisterLayerTx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		wantErr error
	}{
		{
			nameT:   "res == nil",
			wantErr: errors.New("в аргументе w нет указателя"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			var res http.ResponseWriter
			if tt.nameT == "res == nil" {
				res = nil
			} else {
				res = httptest.NewRecorder()
			}

			err := UserRegisterLayerTx(res, "token")
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// UserLogin

func Test_UserLoginLayerRx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT        string
		urlT         string
		bodyT        RegisterRx
		contentTypeT string
	}{
		{
			nameT: "Корректные данные",
			urlT:  "/aaa",
			bodyT: RegisterRx{
				Login:    "AAA",
				Password: "BBB",
			},
			contentTypeT: "application/json",
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			bodyBytes, err := json.Marshal(tt.bodyT)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			req := httptest.NewRequest(http.MethodPost, tt.urlT, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", tt.contentTypeT)

			result, err := UserLoginLayerRx(req)
			require.NoErrorf(t, err, "неожиданная ошибка UserLoginLayerRx: <%v>", err)
			assert.Equalf(t, tt.bodyT.Login, result.Login, "ожидался логин <%s>, а принято <%s>", tt.bodyT.Login, result.Login)
			assert.Equalf(t, tt.bodyT.Password, result.Password, "ожидался пароль <%s>, а принято <%s>", tt.bodyT.Password, result.Password)
		})
	}
}

func Test_UserLoginLayerRx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT       string
		urlT        string
		body        RegisterRx
		contentType string
		wantErr     error
	}{
		{
			nameT: "пустой Login",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "",
				Password: "BBB",
			},
			contentType: "application/json",
			wantErr:     errors.New("400"),
		},
		{
			nameT: "пустой Password",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "AAA",
				Password: "",
			},
			contentType: "application/json",
			wantErr:     errors.New("400"),
		},
		{
			nameT: "Content-Type",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "AAA",
				Password: "BBB",
			},
			contentType: "aaa",
			wantErr:     errors.New("400"),
		},
		{
			nameT: "req == nil",
			urlT:  "/aaa",
			body: RegisterRx{
				Login:    "AAA",
				Password: "BBB",
			},
			contentType: "application/json",
			wantErr:     errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			bodyBytes, err := json.Marshal(tt.body)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			req := httptest.NewRequest(http.MethodPost, tt.urlT, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", tt.contentType)

			if tt.nameT == "req == nil" {
				req = nil
			}

			_, err = UserLoginLayerRx(req)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

func Test_UserLoginLayerTx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT      string
		tokenT     string
		wantErr    error
		wantStatus int
	}{
		{
			nameT:      "Корректные данные",
			tokenT:     "AAA",
			wantErr:    nil,
			wantStatus: http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			err := UserLoginLayerTx(res, tt.tokenT)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.Equalf(t, tt.wantErr, err, "ожидалось <%v> а принято <%v>", tt.wantErr, err)
			assert.Equalf(t, tt.tokenT, res.Header().Get("Authorization"), "ожидался токен <%s> а принято <%s>", tt.tokenT, res.Header().Get("Authorization"))
			assert.Equalf(t, tt.wantStatus, resp.StatusCode, "ожидался код <%d> а принято <%d>", tt.wantStatus, resp.StatusCode)

		})
	}
}

func Test_UserLoginLayerTx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		tokenT  string
		wantErr error
	}{
		{
			nameT:   "res == nil",
			tokenT:  "AAA",
			wantErr: errors.New("в аргументе w нет указателя"),
		},
		{
			nameT:   "нет токена",
			tokenT:  "",
			wantErr: errors.New("в аргументе token нет содержимого"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			var res http.ResponseWriter
			if tt.nameT == "res == nil" {
				res = nil
			} else {
				res = httptest.NewRecorder()
			}

			if tt.nameT == "нет токена" {
				tt.tokenT = ""
			}

			err := UserLoginLayerTx(res, tt.tokenT)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// AddOrder

func Test_AddOrderLayerRx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT        string
		urlT         string
		tokenT       string
		bodyT        string
		contentTypeT string
	}{
		{
			nameT:        "Корректные данные",
			urlT:         "/aaa",
			tokenT:       "aaa",
			bodyT:        "orderAAA",
			contentTypeT: "text/plain",
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			bodyBytes, err := json.Marshal(tt.bodyT)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			req := httptest.NewRequest(http.MethodPost, tt.urlT, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", tt.contentTypeT)
			req.Header.Set("Authorization", tt.tokenT)

			orderRx, tokenRx, err := AddOrderLayerRx(req)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.tokenT, tokenRx, "ожидался токен <%s>, а принято <%s>", tt.tokenT, tokenRx)

			orderRx = strings.Trim(orderRx, "\"")
			assert.Equalf(t, tt.bodyT, orderRx, "ожидался заказ <%s>, а принято <%s>", tt.bodyT, orderRx)
		})
	}
}

func Test_AddOrderLayerRx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT        string
		urlT         string
		tokenT       string
		bodyT        string
		contentTypeT string
		wantErr      error
	}{

		{
			nameT:        "нет токена",
			urlT:         "/aaa",
			tokenT:       "",
			bodyT:        "orderAAA",
			contentTypeT: "text/plain",
			wantErr:      errors.New("401"),
		},
		{
			nameT:        "нет номера заказа",
			urlT:         "/aaa",
			tokenT:       "ccc",
			bodyT:        "",
			contentTypeT: "text/plain",
			wantErr:      errors.New("400"),
		},
		{
			nameT:        "Content-Type",
			urlT:         "/aaa",
			tokenT:       "ccc",
			bodyT:        "",
			contentTypeT: "text",
			wantErr:      errors.New("400"),
		},
		{
			nameT:        "req == nil",
			urlT:         "/aaa",
			tokenT:       "ccc",
			bodyT:        "fff",
			contentTypeT: "text",
			wantErr:      errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			bodyBytes, err := json.Marshal(tt.bodyT)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			req := httptest.NewRequest(http.MethodPost, tt.urlT, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", tt.contentTypeT)
			req.Header.Set("Authorization", tt.tokenT)

			if tt.nameT == "req == nil" {
				req = nil
			}
			_, _, err = AddOrderLayerRx(req)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

func Test_AddOrderLayerTx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT      string
		wantErr    error
		wantStatus int
	}{
		{
			nameT:      "Корректные данные",
			wantErr:    nil,
			wantStatus: http.StatusAccepted,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			err := AddOrderLayerTx(res)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.Equalf(t, tt.wantErr, err, "ожидалось <%v> а принято <%v>", tt.wantErr, err)
			assert.Equalf(t, tt.wantStatus, resp.StatusCode, "ожидался код <%d> а принято <%d>", tt.wantStatus, resp.StatusCode)

		})
	}
}

func Test_AddOrderLayerTx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		wantErr error
	}{
		{
			nameT:   "res == nil",
			wantErr: errors.New("в аргументе w нет указателя"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			var res http.ResponseWriter
			if tt.nameT == "res == nil" {
				res = nil
			} else {
				res = httptest.NewRecorder()
			}

			err := AddOrderLayerTx(res)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// GetOrdersUser

func Test_GetOrdersUserLayerRx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT  string
		urlT   string
		tokenT string
	}{
		{
			nameT:  "Корректные данные",
			urlT:   "/aaa",
			tokenT: "aaa",
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, tt.urlT, nil)
			req.Header.Set("Authorization", tt.tokenT)

			tokenRx, err := GetOrdersUserLayerRx(req)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.tokenT, tokenRx, "ожидался токен <%s>, а принято <%s>", tt.tokenT, tokenRx)
		})
	}
}

func Test_GetOrdersUserLayerRx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		urlT    string
		tokenT  string
		wantErr error
	}{

		{
			nameT:   "нет токена",
			urlT:    "/aaa",
			tokenT:  "",
			wantErr: errors.New("401"),
		},
		{
			nameT:   "req == nil",
			urlT:    "/aaa",
			tokenT:  "ccc",
			wantErr: errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, tt.urlT, nil)
			req.Header.Set("Authorization", tt.tokenT)

			if tt.nameT == "req == nil" {
				req = nil
			}
			_, err := GetOrdersUserLayerRx(req)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

func Test_GetOrdersUserLayerTx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT      string
		ordersT    []actions.Order
		wantStatus int
	}{
		{
			nameT: "Корректные данные",
			ordersT: []actions.Order{
				{
					Number:     "ord1",
					Status:     "AAA",
					Accrual:    100,
					UploadedAt: "2006",
				},
			},
			wantStatus: http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			err := GetOrdersUserLayerTx(res, tt.ordersT)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.NoErrorf(t, err, "неожиданная ошибка запроса <%v>", err)
			require.Equalf(t, tt.wantStatus, resp.StatusCode, "ожидался код <%d> а принято <%d>", tt.wantStatus, resp.StatusCode)

			// Чтение тела ответа
			rxData := make([]actions.Order, 0)

			rxBytes, err := io.ReadAll(resp.Body)
			require.NoErrorf(t, err, "неожиданная ошибка чтения тела ответа <%v>", err)

			err = json.Unmarshal(rxBytes, &rxData)
			require.NoErrorf(t, err, "неожиданная ошибка Unmarshal <%v>", err)

			// Проверка содержимого ответа
			assert.Equalf(t, tt.ordersT[0].Number, rxData[0].Number, "ожидаля Number <%s> а принято <%s>", tt.ordersT[0].Number, rxData[0].Number)
			assert.Equalf(t, tt.ordersT[0].Accrual, rxData[0].Accrual, "ожидаля Accrual <%s> а принято <%s>", tt.ordersT[0].Accrual, rxData[0].Accrual)
			assert.Equalf(t, tt.ordersT[0].Status, rxData[0].Status, "ожидаля Status <%s> а принято <%s>", tt.ordersT[0].Status, rxData[0].Status)
			assert.Equalf(t, tt.ordersT[0].UploadedAt, rxData[0].UploadedAt, "ожидаля UploadedAt <%s> а принято <%s>", tt.ordersT[0].UploadedAt, rxData[0].UploadedAt)
		})
	}
}

func Test_GetOrdersUserLayerTx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		ordersT []actions.Order
		wantErr error
	}{
		{
			nameT:   "Нет данных",
			ordersT: []actions.Order{},
			wantErr: errors.New("500"),
		},
		{
			nameT:   "res == nil",
			ordersT: []actions.Order{},
			wantErr: errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			if tt.nameT == "Нет данных" {
				tt.ordersT = nil
			}

			var res http.ResponseWriter
			if tt.nameT == "res == nil" {
				res = nil
			} else {
				res = httptest.NewRecorder()
			}

			err := GetOrdersUserLayerTx(res, tt.ordersT)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// userBalance

func Test_GetUserBalanceLayerRx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT  string
		urlT   string
		tokenT string
	}{
		{
			nameT:  "Корректные данные",
			urlT:   "/aaa",
			tokenT: "aaa",
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, tt.urlT, nil)
			req.Header.Set("Authorization", tt.tokenT)

			tokenRx, err := GetUserBalanceLayerRx(req)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.tokenT, tokenRx, "ожидался токен <%s>, а принято <%s>", tt.tokenT, tokenRx)
		})
	}
}

func Test_GetUserBalanceLayerRx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		urlT    string
		tokenT  string
		wantErr error
	}{

		{
			nameT:   "нет токена",
			urlT:    "/aaa",
			tokenT:  "",
			wantErr: errors.New("401"),
		},
		{
			nameT:   "req == nil",
			urlT:    "/aaa",
			tokenT:  "ccc",
			wantErr: errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, tt.urlT, nil)
			req.Header.Set("Authorization", tt.tokenT)

			if tt.nameT == "req == nil" {
				req = nil
			}
			_, err := GetUserBalanceLayerRx(req)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

func Test_GetUserBalanceLayerTx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT      string
		balanceT   actions.Balance
		wantStatus int
	}{
		{
			nameT: "Корректные данные",
			balanceT: actions.Balance{
				Current:   10,
				Withdrawn: 9,
			},
			wantStatus: http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			err := GetUserBalanceLayerTx(res, tt.balanceT)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.NoErrorf(t, err, "неожиданная ошибка запроса <%v>", err)
			require.Equalf(t, tt.wantStatus, resp.StatusCode, "ожидался код <%d> а принято <%d>", tt.wantStatus, resp.StatusCode)

			// Чтение тела ответа
			var rxData actions.Balance

			rxBytes, err := io.ReadAll(resp.Body)
			require.NoErrorf(t, err, "неожиданная ошибка чтения тела ответа <%v>", err)

			err = json.Unmarshal(rxBytes, &rxData)
			require.NoErrorf(t, err, "неожиданная ошибка Unmarshal <%v>", err)

			// Проверка содержимого ответа
			assert.Equalf(t, tt.balanceT.Current, rxData.Current, "ожидаля Current <%s> а принято <%s>", tt.balanceT.Current, rxData.Current)
			assert.Equalf(t, tt.balanceT.Withdrawn, rxData.Withdrawn, "ожидаля Withdrawn <%s> а принято <%s>", tt.balanceT.Withdrawn, rxData.Withdrawn)
		})
	}
}

func Test_GetUserBalanceLayerTx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT    string
		balanceT actions.Balance
		wantErr  error
	}{
		{
			nameT:    "res == nil",
			balanceT: actions.Balance{},
			wantErr:  errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			var res http.ResponseWriter
			if tt.nameT == "res == nil" {
				res = nil
			} else {
				res = httptest.NewRecorder()
			}

			err := GetUserBalanceLayerTx(res, tt.balanceT)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

// BalanceWithdraw

func Test_BalanceWithdrawLayerRx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT  string
		urlT   string
		bodyT  actions.BalanceWithdraw
		tokenT string
	}{
		{
			nameT: "Корректные данные",
			urlT:  "/aaa",
			bodyT: actions.BalanceWithdraw{
				Order: "aaa",
				Sum:   111,
			},
			tokenT: "aaa",
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			bodyBytes, err := json.Marshal(tt.bodyT)
			require.NoErrorf(t, err, "неожиданная ошибка Marshal: <%v>", err)

			req := httptest.NewRequest(http.MethodPost, tt.urlT, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Authorization", tt.tokenT)

			tokenRx, balanceRx, err := BalanceWithdrawLayerRx(req)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.tokenT, tokenRx, "ожидался токен <%s>, а принято <%s>", tt.tokenT, tokenRx)
			assert.Equalf(t, tt.bodyT.Order, balanceRx.Order, "ожидался Order <%s>, а принято <%s>", tt.bodyT.Order, balanceRx.Order)
			assert.Equalf(t, tt.bodyT.Sum, balanceRx.Sum, "ожидался Sum <%s>, а принято <%s>", tt.bodyT.Sum, balanceRx.Sum)
		})
	}
}

func Test_BalanceWithdrawLayerRx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		urlT    string
		tokenT  string
		wantErr error
	}{

		{
			nameT:   "нет токена",
			urlT:    "/aaa",
			tokenT:  "",
			wantErr: errors.New("401"),
		},
		{
			nameT:   "req == nil",
			urlT:    "/aaa",
			tokenT:  "ccc",
			wantErr: errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, tt.urlT, nil)
			req.Header.Set("Authorization", tt.tokenT)

			if tt.nameT == "req == nil" {
				req = nil
			}
			_, _, err := BalanceWithdrawLayerRx(req)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

func Test_BalanceWithdrawLayerTx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT      string
		wantErr    error
		wantStatus int
	}{
		{
			nameT:      "Корректные данные",
			wantErr:    nil,
			wantStatus: http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			err := BalanceWithdrawLayerTx(res)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.Equalf(t, tt.wantErr, err, "ожидалось <%v> а принято <%v>", tt.wantErr, err)
			assert.Equalf(t, tt.wantStatus, resp.StatusCode, "ожидался код <%d> а принято <%d>", tt.wantStatus, resp.StatusCode)
		})
	}
}

func Test_BalanceWithdrawLayerTx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		wantErr error
	}{
		{
			nameT:   "res == nil",
			wantErr: errors.New("в аргументе w нет указателя"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			err := BalanceWithdrawLayerTx(nil)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)

		})
	}
}

// GetHistoryWithdrawels

func Test_GetHistoryWithdrawelsLayerRx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT  string
		urlT   string
		tokenT string
	}{
		{
			nameT:  "Корректные данные",
			urlT:   "/aaa",
			tokenT: "aaa",
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, tt.urlT, nil)
			req.Header.Set("Authorization", tt.tokenT)

			tokenRx, err := GetHistoryWithdrawelsLayerRx(req)
			require.NoErrorf(t, err, "неожиданная ошибка: <%v>", err)
			assert.Equalf(t, tt.tokenT, tokenRx, "ожидался токен <%s>, а принято <%s>", tt.tokenT, tokenRx)
		})
	}
}

func Test_GetHistoryWithdrawelsLayerRx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		urlT    string
		tokenT  string
		wantErr error
	}{
		{
			nameT:   "нет токена",
			urlT:    "/aaa",
			tokenT:  "",
			wantErr: errors.New("401"),
		},
		{
			nameT:   "req == nil",
			urlT:    "/aaa",
			tokenT:  "ccc",
			wantErr: errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			req := httptest.NewRequest(http.MethodPost, tt.urlT, nil)
			req.Header.Set("Authorization", tt.tokenT)

			if tt.nameT == "req == nil" {
				req = nil
			}
			_, err := GetHistoryWithdrawelsLayerRx(req)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)
		})
	}
}

func Test_GetHistoryWithdrawelsLayerTx_SUCCESS(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT      string
		ordersT    []WithdrawalResponse
		wantStatus int
	}{
		{
			nameT: "Корректные данные",
			ordersT: []WithdrawalResponse{
				{
					Order:       "aaa",
					Sum:         111,
					ProcessedAt: "2999",
				},
			},
			wantStatus: http.StatusOK,
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			err := GetHistoryWithdrawelsLayerTx(res, tt.ordersT)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			require.NoErrorf(t, err, "неожиданная ошибка запроса <%v>", err)
			require.Equalf(t, tt.wantStatus, resp.StatusCode, "ожидался код <%d> а принято <%d>", tt.wantStatus, resp.StatusCode)

			// Чтение тела ответа
			rxData := make([]WithdrawalResponse, 0)

			rxBytes, err := io.ReadAll(resp.Body)
			require.NoErrorf(t, err, "неожиданная ошибка чтения тела ответа <%v>", err)

			err = json.Unmarshal(rxBytes, &rxData)
			require.NoErrorf(t, err, "неожиданная ошибка Unmarshal <%v>", err)

			// Проверка содержимого ответа
			assert.Equalf(t, tt.ordersT[0].Order, rxData[0].Order, "ожидаля Order <%s> а принято <%s>", tt.ordersT[0].Order, rxData[0].Order)
			assert.Equalf(t, tt.ordersT[0].Sum, rxData[0].Sum, "ожидаля Sum <%s> а принято <%s>", tt.ordersT[0].Sum, rxData[0].Sum)
			assert.Equalf(t, tt.ordersT[0].ProcessedAt, rxData[0].ProcessedAt, "ожидаля ProcessedAt <%s> а принято <%s>", tt.ordersT[0].ProcessedAt, rxData[0].ProcessedAt)
		})
	}

}

func Test_GetHistoryWithdrawelsLayerTx_FAULT(t *testing.T) {

	// Данные для тестов
	testsData := []struct {
		nameT   string
		ordersT []WithdrawalResponse
		wantErr error
	}{
		{
			nameT:   "Нет данных",
			ordersT: []WithdrawalResponse{},
			wantErr: errors.New("500"),
		},
	}

	// Тесты
	for _, tt := range testsData {
		t.Run(tt.nameT, func(t *testing.T) {

			res := httptest.NewRecorder()

			if tt.nameT == "Нет данных" {
				tt.ordersT = nil
			}

			err := GetHistoryWithdrawelsLayerTx(res, tt.ordersT)
			assert.Equalf(t, tt.wantErr, err, "ожидалась ошибка <%v> а принято <%v>", tt.wantErr, err)

			resp := res.Result()
			defer func() {
				_ = resp.Body.Close()
			}()
		})
	}
}
