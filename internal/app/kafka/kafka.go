package kafka

import (
	"github.com/IBM/sarama"
	tele "gopkg.in/telebot.v4"
	"ibTgBot/configs"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Kafka struct {
	reUrlId *regexp.Regexp
	reTagId *regexp.Regexp
	re429   *regexp.Regexp // Ошибка 429 ТГ лимит отправки сообщений
	EsTopic string
	RuTopic string
	EsCanal int64
	RuCanal int64
	s       Service
	d       DB
}

type Service interface {
	GetBot() *tele.Bot
	GetTG() configs.TgConfig
}

type DB interface {
	SetMsgId(msgId int, urlId, clientId string)
	GetSubscribers(tagId, lang string) ([]int, error)
}

func New(s Service, d DB) *Kafka {
	return &Kafka{
		reUrlId: regexp.MustCompile(`\(urlId: (\d+)\)`),
		reTagId: regexp.MustCompile(`\(tagId: (\d+)\)`),
		re429:   regexp.MustCompile(`retry after \d+ \(429\)`),
		EsTopic: "esInfobot",
		RuTopic: "ruInfobot",
		EsCanal: s.GetTG().EsCanal,
		RuCanal: s.GetTG().RuCanal,
		s:       s, d: d,
	}
}

func (k *Kafka) Run() {
	go k.KafkaRead("es", k.EsTopic, k.EsCanal)
	go k.KafkaRead("ru", k.RuTopic, k.RuCanal)
}

func (k *Kafka) KafkaConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	return config
}

func (k *Kafka) KafkaClient() sarama.Client {
	client, err := sarama.NewClient([]string{"localhost:9092"}, k.KafkaConfig())
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %s", err)
	}
	return client
}

func (k *Kafka) KafkaConsumer(client sarama.Client) sarama.Consumer {
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		log.Fatalf("Failed to start consumer: %s", err)
	}
	return consumer
}

func (k *Kafka) KafkaOffsetManager(client sarama.Client) sarama.OffsetManager {
	offsetManager, err := sarama.NewOffsetManagerFromClient("consumerGroup", client) //ibTgClient
	if err != nil {
		log.Fatalf("Failed to create offset manager: %s", err)
	}
	return offsetManager

}

func (k *Kafka) SendMsg(messageBody string, telegramChannel int64) (int, error) {
	for attempts := 1; attempts <= 3; attempts++ { // Повторяем попытку отправки сообщения 3 раза
		msg, err := k.s.GetBot().Send(tele.ChatID(telegramChannel),
			messageBody,
			&tele.SendOptions{
				ParseMode:           tele.ModeHTML,
				DisableNotification: true,
			})

		if err != nil {
			if attempts < 3 && k.re429.MatchString(err.Error()) {
				log.Printf("Failed to send message to Telegram: %s, retrying in %d seconds...", err, 2*attempts)
				time.Sleep(time.Duration(2*attempts) * time.Second)
				continue
			}
			log.Printf("Failed to send message to Telegram after %d attempts: %s", attempts, err)
			log.Printf("Message: %s", messageBody)
		} else {
			//go k.d.SetMsgId(msg.ID, urlId, clientId)
			log.Printf("Message sent to Telegram channel: %s, urlId: %s",
				strconv.FormatInt(telegramChannel, 10))
			return msg.ID, nil
		}
		time.Sleep(time.Duration(1*attempts) * time.Second)
	}
	return -1, nil
}

func (k *Kafka) SendSubscribers(tagId, lang, message string) {
	subscribers, err := k.d.GetSubscribers(tagId, lang)
	if err != nil {
		log.Printf("Failed to get subscribers: %s", err)
		return
	}

	if len(subscribers) > 0 {
		for _, subscriber := range subscribers {
			go k.SendMsg(message, int64(subscriber))
		}
	}
}

func (k *Kafka) SendToTelegram(message, clientId string, telegramChannel int64) {
	urlIdMatch := k.reUrlId.FindStringSubmatch(message)
	tagIdMatch := k.reTagId.FindStringSubmatch(message)
	urlId := urlIdMatch[1]
	messageBody := strings.TrimSpace(strings.Replace(message, urlIdMatch[0], "", 1))

	if len(tagIdMatch) > 1 {
		tagId := tagIdMatch[1]
		go k.SendSubscribers(tagId, clientId, messageBody)
	}

	msgId, err := k.SendMsg(messageBody, telegramChannel)
	if err != nil || msgId == -1 {
		log.Printf("Ошибка отправки сообщения Телеграмм: %s", err)
	} else {
		// Помечаю сообщение как отправленное и присваиваю номер
		go k.d.SetMsgId(msgId, urlId, clientId)
	}
}

func (k *Kafka) KafkaRead(clientId, topic string, telegramChannel int64) {
	log.Printf("Initializing %s Kafka consumer...", clientId)
	// Создаем нового клиента
	client := k.KafkaClient()
	defer client.Close()

	// Создаем нового потребителя
	consumer := k.KafkaConsumer(client)
	defer consumer.Close()

	// Создаем менеджер смещений
	offsetManager := k.KafkaOffsetManager(client)
	defer offsetManager.Close()

	// Создаем менеджер смещений для каждой партиции
	partitionOffsetManager, err := offsetManager.ManagePartition(topic, 0)
	if err != nil {
		log.Fatalf("Failed to create partition offset manager: %s", err)
	}
	defer partitionOffsetManager.Close()

	// Читаем текущее смещение из OffsetManager
	offset, _ := partitionOffsetManager.NextOffset()
	log.Printf("Starting at offset: %d", offset)
	if offset == sarama.OffsetNewest || offset == sarama.OffsetOldest {
		offset = sarama.OffsetOldest // Начнем с самого старого сообщения, если нет сохраненного смещения
	}

	log.Printf("Subscribed to Kafka topic %s...", topic)
	// Подписываемся на топик с использованием текущего смещения
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, offset)
	if err != nil {
		log.Fatalf("Failed to start partition consumer: %s", err)
	}
	defer partitionConsumer.Close()

	// Чтение сообщений в цикле
	for {
		log.Println("Waiting for messages from Kafka...")
		select {
		case msg := <-partitionConsumer.Messages():
			log.Println("Sending message to Telegram channel...")
			message := string(msg.Value)
			// Отправка сообщения в Telegram
			go k.SendToTelegram(message, clientId, telegramChannel)
			// Коммит смещения
			partitionOffsetManager.MarkOffset(msg.Offset+1, "")

		case err := <-partitionConsumer.Errors():
			log.Printf("Error: %s", err)
		}
	}
}
