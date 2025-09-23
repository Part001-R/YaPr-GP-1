package flags

import (
	"flag"
	"os"
)

type FlagsT struct {
	RunAddress           string
	DatabaseURI          string
	AccuralSystemAddress string
}

func ParseFlags() FlagsT {

	var flags = FlagsT{}

	flag.StringVar(&flags.RunAddress, "a", ":8888", "адрес и порт сервера")
	flag.StringVar(&flags.DatabaseURI, "d", "", "адрес подключения к БД")
	flag.StringVar(&flags.AccuralSystemAddress, "r", "http://localhost:8080", "адрес системы расчёта начислений")

	flag.Parse()

	if envValue := os.Getenv("RUN_ADDRESS"); envValue != "" {
		flags.RunAddress = envValue
	}
	if envValue := os.Getenv("DATABASE_URI"); envValue != "" {
		flags.DatabaseURI = envValue
	}
	if envValue := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envValue != "" {
		flags.AccuralSystemAddress = envValue
	}

	return flags
}
