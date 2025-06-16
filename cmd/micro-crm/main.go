package main

import (
	"github.com/joho/godotenv"
	"log"
	"micro-CRM/internal/api"
	"micro-CRM/internal/models"
	"os"
)

var (
	customVars models.EnvParams
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file")
	}
	customVars.DbPath = os.Getenv("DB_PATH")
	customVars.JWTToken = os.Getenv("JWT_TOKEN")
	customVars.ApiPort = os.Getenv("API_PORT")
	if customVars.ApiPort == "" {
		customVars.ApiPort = "9080"
	}
	if customVars.DbPath == "" {
		customVars.DbPath = "./database/micro-crm.db"
	}
	if customVars.JWTToken == "" {
		log.Fatalln("JWT_TOKEN environment variable must be set")
	}
	serverApi := api.Api{
		Params: customVars,
	}
	serverApi.Start()
}
