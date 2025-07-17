package main

import (
	"github.com/joho/godotenv"
	"log"
	"micro-CRM/internal/api"
	"micro-CRM/internal/models"
	"os"
	"os/exec"
)

var (
	customVars models.EnvParams
)

func main() {
	_ = godotenv.Load()
	customVars.JWTToken = os.Getenv("JWT_TOKEN")
	customVars.ApiPort = os.Getenv("API_PORT")
	customVars.WebUiUrl = os.Getenv("WEB_UI_BASE_URL")
	customVars.DataPath = os.Getenv("DATA_STORAGE_PATH")
	customVars.DbPath = customVars.DataPath + "/database/micro-crm.db"
	customVars.CertFilePath = os.Getenv("CERT_FILE_PATH")
	customVars.KeyFilePath = os.Getenv("KEY_FILE_PATH")

	_, err := os.Stat(customVars.DataPath)
	if os.IsNotExist(err) {
		log.Println("Data storage path env variable missing, trying defaults")
		customVars.DbPath = customVars.DataPath + "/database/micro-crm.db"
	}
	if customVars.ApiPort == "" {
		customVars.ApiPort = models.DefaultApiPort
	}
	if customVars.DbPath == "" {
		customVars.DbPath = customVars.DataPath + "/database/micro-crm.db"
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
		log.Println("Cert file and key file path env variable missing, trying defaults")
		_, certErr := os.Stat(customVars.DataPath + "/certs/cert.pem")
		_, keyErr := os.Stat(customVars.DataPath + "/certs/key.pem")
		if os.IsNotExist(certErr) && os.IsNotExist(keyErr) {
			log.Println("Key and cert files do not exist: auto-generating new ones")
			cdm := exec.Command("bash", "./autogen-key.sh")
			err := cdm.Run()
			if err != nil {
				log.Fatalln("Auto cert generation failed", err)
			}
		}
		customVars.KeyFilePath = customVars.DataPath + "/certs/key.pem"
		customVars.CertFilePath = customVars.DataPath + "/certs/cert.pem"
	}
	serverApi := api.NewApi(customVars)
	serverApi.Start()
}
