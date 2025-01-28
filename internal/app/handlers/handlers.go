package handlers

import (
	"encoding/json"
	"fmt"
	tele "gopkg.in/telebot.v4"
	"ibTgBot/internal/app/db"
	"log"
	"strconv"
)

var (
	menu    = &tele.ReplyMarkup{}
	btnPrev = menu.Data("⬅", "prev")
	btnNext = menu.Data("➡", "next")

	//btnYes = menu.Data("Да", "yes")
	//btnNo  = menu.Data("Нет", "no")
	// массив состояний кнопок
	buttonStates = make(map[int]string) // Используем карту для хранения состояния кнопок
	limit        = 99
)

type Handlers struct {
	s             Service
	d             DB
	userId        int64
	langSelected  string
	userUserName  string
	userFirstName string
	userLastName  string
}

type Service interface {
	GetBot() *tele.Bot
	Run() error
	GetBotName() string
	GetBotReady() chan struct{}
}

type DB interface {
	ReadTags(limit int, active bool, lang string) ([]db.Tag, error)
	CreateUser(userId int64, userName, firstName, lastName, lang string) error
	ManageCategories(userId int64, tagId *int) (string, error)
}

func New(s Service, d DB) *Handlers {
	return &Handlers{s: s, d: d, langSelected: "ru"}
}

func (h *Handlers) Run() error {
	err := h.s.Run()
	if err != nil {
		return err
	}

	return nil
}

func (h *Handlers) SetupHandlers() {
	b := h.s.GetBot()

	// Команда /subscribe для отправки меню
	b.Handle("/subscribe", func(c tele.Context) error {
		// Автоопределение языка пользователя
		userLang := c.Sender().LanguageCode
		userId := c.Sender().ID
		// Данные пользователя
		h.userId = userId
		h.userUserName = c.Sender().Username
		h.userFirstName = c.Sender().FirstName
		h.userLastName = c.Sender().LastName

		// Запрос подтверждения у пользователя
		btnYes := menu.Data("Да", "yes")
		btnNo := menu.Data("Нет", "no")
		menu.Inline(
			menu.Row(btnYes, btnNo),
		)
		msg, err := b.Send(tele.ChatID(userId), fmt.Sprintf("Язык определен как '%s'. Продолжить?", userLang), menu)
		if err != nil {
			return err
		}
		// Привязка обработчиков к кнопкам подтверждения и отмены
		b.Handle(&btnYes, func(c tele.Context) error {
			h.langSelected = "ru"
			b.Delete(msg)
			return h.HandleConfirmation(c, "ru")
		})
		b.Handle(&btnNo, func(c tele.Context) error {
			h.langSelected = "es"
			b.Delete(msg)
			return h.HandleConfirmation(c, "es")
		})
		return nil
	})

	// Обработка нажатия кнопки "Подписаться"
	b.Handle(&btnNext, func(c tele.Context) error {
		c.Respond()
		return c.Send("Вы подписаны!")
	})

	// Обработка нажатия кнопки "Отписаться"
	b.Handle(&btnPrev, func(c tele.Context) error {
		c.Respond()
		return c.Send("Вы отписаны!")
	})
}

func (h *Handlers) HandleConfirmation(c tele.Context, userLang string) error {
	//Пользователь ответил, создаю пользователя в БД
	err := h.d.CreateUser(h.userId, h.userUserName, h.userFirstName, h.userLastName, userLang)
	if err != nil {
		log.Printf("Ошибка при создании пользователя %e", err)
	}
	// Получаем данные из SQLReadTags
	tags, err := h.d.ReadTags(limit, true, userLang)
	if err != nil {
		log.Printf("Ошибка при создании меню тегов %e", err)
		return c.Send("Ошибка при создании меню тегов")
	}

	// Инициализация состояния кнопок на основе полученных данных
	for _, tag := range tags {
		buttonStates[tag.ID] = "0" // Все кнопки неактивны по умолчанию
	}

	// Создание меню на основе состояний кнопок
	h.CreateButtons(tags)
	// Отправка меню с кнопками
	return c.Send("Выберите опцию:", menu)
}

func (h *Handlers) CreateButtons(tags []db.Tag) {
	var buttons []tele.Row
	userCats, err := h.d.ManageCategories(h.userId, nil)
	if err != nil {
		log.Printf("Ошибка при получении категорий пользователя: %e", err)
	}

	var userCatsSet []int
	// Заполнение слайса id категорий пользователя для быстрой проверки
	err = json.Unmarshal([]byte(userCats), &userCatsSet)
	if err != nil {
		log.Printf("Ошибка при парсинге категорий пользователя: %e", err)
	}

	// Обновляем buttonStates
	buttonStates := make(map[int]string)
	for _, cat := range userCatsSet {
		buttonStates[cat] = "1"
	}

	for i := 0; i < len(tags); i += 2 {
		btn1ID := tags[i].ID
		log.Printf("TagId: %d", btn1ID)
		btn1Text := "❌ " + tags[i].Value

		// Если btn1ID находится в категориях пользователя, то помечаем кнопку как активную
		if buttonStates[btn1ID] == "1" {
			btn1Text = "✅ " + tags[i].Value
		} else {
			buttonStates[btn1ID] = "0"
		}

		btn1 := menu.Data(btn1Text, "btn_"+strconv.Itoa(btn1ID), strconv.Itoa(btn1ID))

		var btn2 tele.Btn
		var btn2ID int
		if i+1 < len(tags) {
			btn2ID = tags[i+1].ID
			btn2Text := "❌ " + tags[i+1].Value

			if buttonStates[btn2ID] == "1" {
				btn2Text = "✅ " + tags[i+1].Value
			} else {
				buttonStates[btn2ID] = "0"
			}

			btn2 = menu.Data(btn2Text, "btn_"+strconv.Itoa(btn2ID), strconv.Itoa(btn2ID))
			buttons = append(buttons, menu.Row(btn1, btn2))
		} else {
			buttons = append(buttons, menu.Row(btn1))
		}

		h.AddButtonHandler(btn1, btn1ID)
		if i+1 < len(tags) {
			h.AddButtonHandler(btn2, btn2ID)
		}
	}

	menu.Inline(buttons...)
}

func (h *Handlers) AddButtonHandler(btn tele.Btn, id int) {
	h.s.GetBot().Handle(&btn, func(c tele.Context) error {
		// Обновление состояния кнопки
		switch buttonStates[id] {
		case "0":
			buttonStates[id] = "1"
		case "1":
			buttonStates[id] = "0"
		}

		_, err := h.d.ManageCategories(h.userId, &id)
		if err != nil {
			log.Fatalf("Ошибка при обновлении категорий пользователя %e", err)
		}

		// Пересоздание всех кнопок с обновленными значениями
		tags, err := h.d.ReadTags(limit, true, h.langSelected)
		if err != nil {
			return c.Send("Ошибка обновления меню тагов")
		}

		h.CreateButtons(tags)
		c.Respond()
		return c.EditOrReply(menu)
	})
}
