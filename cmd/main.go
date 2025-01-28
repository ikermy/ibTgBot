package main

import (
	"ibTgBot/configs"
	"ibTgBot/internal/pkg/app"
	"log"
)

func main() {
	conf, err := configs.New("./configs/.env")
	if err != nil {
		log.Fatal(err)
	}

	err = app.New(conf).Run()
	if err != nil {
		log.Fatal(err)
	}
}
