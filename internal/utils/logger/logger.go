package logger

import (
	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(level string) error {

	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	// создаём новую конфигурацию логера
	cfg := zap.NewProductionConfig()

	// устанавливаем уровень
	cfg.Level = lvl

	// создаём логер на основе конфигурации
	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	// устанавливаем синглтон
	Log = zl
	return nil
}

// Вспомогательная функция для отладки работы приложения.
//
// Параметры:
//
// str - строка, для записи в файл.

/*
func WriteInFileDebugData(str string) {
	filename := "debug.txt"

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("ошибка <%v> открытия файла <%s>", err, filename)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.WriteString(str + "\n"); err != nil {
		log.Fatalf("ошибка <%v> записи в файл <%s>", err, filename)
	}
}
*/
