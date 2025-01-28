package init

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log"
	"os"
)

const logPatch = "\"/var/log/info-bot/ibTgBot/ibTgBot.log\""

func init() {
	logFile := &lumberjack.Logger{
		Filename:   logPatch, // Путь к файлу логов
		MaxSize:    1,        // Максимальный размер файла в мегабайтах
		MaxBackups: 5,        // Максимальное количество старых файлов для хранения
		MaxAge:     28,       // Максимальное количество дней для хранения старых файлов
		Compress:   true,     // Сжимать ли старые файлы
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile) // Включаем временные метки и короткие имена файлов
}
