package configs

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
)

type ConfIn interface {
	GetDB() DbConfig
	GetTG() TgConfig
}

type DBIn interface {
	name() string
}

type TgIn interface {
	GetToken() string
}

func (c *Conf) GetDB() DbConfig {
	return c.DB
}

func (c *Conf) GetTG() TgConfig {
	return c.TG
}

type Conf struct {
	DB DbConfig
	TG TgConfig
}

type DbConfig struct {
	Name     string `mapstructure:"name"`
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type TgConfig struct {
	Token   string `mapstructure:"token"`
	RuCanal int64  `mapstructure:"ruCanal"`
	EsCanal int64  `mapstructure:"esCanal"`
}

func New(confPatch string) (*Conf, error) {
	// Получаем текущую рабочую директорию
	wd, erra := os.Getwd()
	if erra != nil {
		return nil, fmt.Errorf("ошибка получени текущего каталога: %v\n", erra)
	}

	// Формируем путь к файлу конфигурации
	configPath := wd + confPatch

	// Проверяем, существует ли файл по указанному пути
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Файл конфигурации не существует: %s\n", configPath)
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("hcl") // Указываем формат файла конфигурации

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("ошибка чтения файла конфигурации: %w", err)
	}

	// Извлекаем массив карт для БД
	var configs []map[string]interface{}
	if err := viper.UnmarshalKey("db", &configs); err != nil {
		return nil, fmt.Errorf("невозможно прочитать структуру файла конфигурации БД: %w", err)
	}

	var (
		dbConf DbConfig
		tgConf TgConfig
	)
	// Предполагаем, что первая карта в массиве содержит наши настройки
	if len(configs) > 0 {
		dbConf.Name = configs[0]["name"].(string)
		dbConf.Host = configs[0]["host"].(string)
		dbConf.User = configs[0]["user"].(string)
		dbConf.Password = configs[0]["password"].(string)
	} else {
		return nil, fmt.Errorf("не найдены параметры конфигурации базы данных")
	}

	// Извлекаем массив карт для ТГ
	var tgConfigs []map[string]interface{}
	if err := viper.UnmarshalKey("tg", &tgConfigs); err != nil {
		return nil, fmt.Errorf("невозможно прочитать структуру файла конфигурации ТГ: %w", err)
	}

	// Предполагаем, что первая карта в массиве содержит наши настройки
	if len(tgConfigs) > 0 {
		tgConf.Token = tgConfigs[0]["token"].(string)
		tgConf.RuCanal = int64(tgConfigs[0]["ruCanal"].(int))
		tgConf.EsCanal = int64(tgConfigs[0]["esCanal"].(int))
	} else {
		return nil, fmt.Errorf("не найдены параметры конфигурации ТГ")
	}

	return &Conf{DB: dbConf, TG: tgConf}, nil
}
