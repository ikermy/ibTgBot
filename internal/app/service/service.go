package service

import (
	"ibTgBot/configs"
	"time"

	tele "gopkg.in/telebot.v4"
)

type Service struct {
	botReady chan struct{}
	b        *tele.Bot
	tgConf   configs.TgConfig
}

func New(conf *configs.Conf) *Service {
	return &Service{
		botReady: make(chan struct{}), b: nil, tgConf: conf.GetTG(),
	}
}

type GetBotIn interface {
	GetBot() *tele.Bot
}

type GetBotNameIn interface {
	GetBotName() string
}

type GetTgConfigIn interface {
	GetTG() configs.TgConfig
}

func (s *Service) GetBot() *tele.Bot {
	return s.b
}

func (s *Service) GetTG() configs.TgConfig {
	return s.tgConf
}

func (s *Service) Run() error {
	b, err := tele.NewBot(tele.Settings{
		Token:  s.tgConf.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return err
	}
	s.b = b

	// Закрытие канала уведомления о готовности бота
	close(s.botReady)

	s.b.Start()

	return nil
}

func (s *Service) GetBotReady() chan struct{} {
	return s.botReady
}

func (s *Service) GetBotName() string {
	return s.b.Me.Username
}
