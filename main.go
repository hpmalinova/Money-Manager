package main

import (
	"fmt"
	"github.com/hpmalinova/Money-Manager/rest"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func main() {
	fmt.Println("Staring Service ...")

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	dbname := os.Getenv("DBNAME")

	a := rest.App{}
	a.Init(user, password, dbname)
	a.Run(port)
}
