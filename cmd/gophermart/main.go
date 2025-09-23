package main

import (
	"github.com/Part001-R/YaPr-GP-1/internal/service"
	"github.com/Part001-R/YaPr-GP-1/internal/utils/logger"
	"go.uber.org/zap"
)

func main() {

	err := service.App()
	if err != nil {
		logger.Log.Error("Работа приложения остановлена",
			zap.Error(err),
		)
	}
}
