package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Secrets ...
type Secrets struct {
	GoogleClientID     string `json:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `json:"GOOGLE_CLIENT_SECRET"`
	SessionKey         string `json:"SESSION_KEY"`
	MongoURI           string `json:"MONGO_URI"`
	DatabaseName       string `json:"DATABASE_NAME"`
	AdminEmails       []string `json:"ADMIN_EMAILS"`
	DomainName 				 string `json:"DOMAIN_NAME"`
}

// GetSecrets - gets the secrets from Config.Dev file
func GetSecrets(isDev bool, mySecrets *Secrets) {
	// Checking if in Development mode or not
	isDev = os.Getenv("isDev") == "true"
	fmt.Println("isDev: ", isDev)
	
	// JSON file location
	var filename string
	if isDev {
		filename = "config/config.development.json"
	} else {
		filename = "config/config.production.json"
	}

	// JSON file
	jsonFile, readErr := os.Open(filename)
	if readErr != nil {
		fmt.Println("Error:", readErr)
	}

	defer jsonFile.Close()

	// Reading JSON file
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// Saving data in struct
	marshErr := json.Unmarshal([]byte(byteValue), &mySecrets)
	if marshErr != nil {
		fmt.Println("Error:", marshErr)
	}
}
