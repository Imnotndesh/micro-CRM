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
	customVars.CertFilePath = os.Getenv("CERT_FILE_PATH")
	customVars.KeyFilePath = os.Getenv("KEY_FILE_PATH")
	if customVars.ApiPort == "" {
		customVars.ApiPort = models.DefaultApiPort
	}
	if customVars.DbPath == "" {
		customVars.DbPath = models.DefaultDBPath
	}
	if customVars.JWTToken == "" {
		log.Fatalln("JWT_TOKEN environment variable must be set")
	}
	if customVars.CertFilePath != "" && customVars.KeyFilePath != "" {
		_, certErr := os.Stat(customVars.CertFilePath)
		_, keyErr := os.Stat(customVars.KeyFilePath)
		if os.IsNotExist(certErr) && os.IsNotExist(keyErr) {
			log.Fatalln("Key and cert files do not exist")
		}
	} else {
		log.Fatalln("Cert file and key file path env variable must be passed")
	}
	serverApi := api.NewApi(customVars)
	serverApi.Start()
}
