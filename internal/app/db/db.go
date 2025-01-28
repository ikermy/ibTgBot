package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"ibTgBot/configs"
	"log"
	"strconv"
	"strings"
)

type DB struct {
	dsn string
}

type Tag struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

type GetSubscribersIn interface {
	GetSubscribers(tagId, lang string) ([]int, error)
}

type CreateUserIn interface {
	CreateUser(userId int64, userName, firstName, lastName, lang string) error
}

type ReadTagsIn interface {
	SQLReadTags(limit int, mainTag bool, lang string) ([]Tag, error)
}

type SetMsgIdIn interface {
	SQLSetMsgId(msgId int, urlId, lang string)
}

type ManageCategoriesIn interface {
	ManageCategories(userId int64, tagId *int) (string, error)
}

func New(conf *configs.Conf) *DB {
	return &DB{
		dsn: fmt.Sprintf("%s:%s@tcp(%s)/%s", conf.GetDB().User, conf.GetDB().Password, conf.GetDB().Host, conf.GetDB().Name),
	}
}

func (d *DB) SQLSetMsgId(msgId int, urlId, lang string) {
	urlIdInt, err := strconv.Atoi(urlId)
	if err != nil {
		log.Printf("Failed to convert urlId to int: %s", err)
		return
	}

	db, err := sql.Open("mysql", d.dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Вызов хранимой процедуры SetTgMsg
	_, err = db.Exec("CALL SetTgMsg(?, ?, ?)", lang, urlIdInt, msgId)
	if err != nil {
		log.Printf("Failed to call stored procedure: %s", err)
		return
	}

	log.Println("Set Tg msgId successfully for urlId:", urlId)
}

func (d *DB) SQLReadTags(limit int, mainTag bool, lang string) ([]Tag, error) {
	db, err := sql.Open("mysql", d.dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var result string
	err = db.QueryRow("SELECT ib_tg_ReadTags(?, ?, ?)", limit, mainTag, lang).Scan(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to call stored function: %w", err)
	}

	var tags []Tag
	err = json.Unmarshal([]byte(result), &tags)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return tags, nil
}

func (d *DB) CreateUser(userId int64, userName, firstName, lastName, lang string) error {
	db, err := sql.Open("mysql", d.dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec("CALL ib_tg_CreateUser(?, ?, ?, ?, ?)", userId, userName, firstName, lastName, lang)
	if err != nil {
		return fmt.Errorf("failed to call stored procedure: %w", err)
	}

	return nil
}

func (d *DB) ManageCategories(userId int64, tagId *int) (string, error) {
	db, err := sql.Open("mysql", d.dsn)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var categories string

	if tagId == nil {
		// Если tagId не передан, возвращаем массив категорий
		err = db.QueryRow("CALL ib_tg_ManageCategories(?, NULL)", userId).Scan(&categories)
		if err != nil {
			return "", fmt.Errorf("failed to call stored procedure: %w", err)
		}
	} else {
		// Если tagId передан, добавляем или удаляем его
		_, err = db.Exec("CALL ib_tg_ManageCategories(?, ?)", userId, *tagId)
		if err != nil {
			return "", fmt.Errorf("failed to call stored procedure: %w", err)
		}
	}

	return categories, nil
}

func (d *DB) GetSubscribers(tagId, lang string) ([]int, error) {
	db, err := sql.Open("mysql", d.dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var result string
	err = db.QueryRow("SELECT ib_tg_GetSubscribers(?, ?)", tagId, lang).Scan(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to call stored function: %w", err)
	}

	// Если результат пуст, вернем пустой срез
	if result == "" {
		return []int{}, nil
	}

	// Разделим строку и конвертируем в срез int
	stringIds := strings.Split(result, ",")
	subscribers := make([]int, len(stringIds))
	for i, idStr := range stringIds {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert id to int: %w", err)
		}
		subscribers[i] = id
	}

	return subscribers, nil
}
