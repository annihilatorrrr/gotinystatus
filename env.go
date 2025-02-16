package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gomarkdown/markdown"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CheckInterval     int    `env:"CHECK_INTERVAL, default=60"`
	MaxHistoryEntries int    `env:"MAX_HISTORY_ENTRIES, default=10"`
	ChecksFile        string `env:"CHECKS_FILE, default=checks.yaml"`
	IncidentsFile     string `env:"INCIDENTS_FILE, default=incidents.html"`
	HistoryFile       string `env:"HISTORY_FILE, default=history.json"`
	Port              int    `env:"PORT"`
	Token             string `env:"TOKEN"`
	Chatid            string `env:"CHATID"`
}

func readEnv() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file, falling back to default environment variables")
	}
	var ret Config
	if err := envconfig.Process(context.Background(), &ret); err != nil {
		log.Fatal(err)
	}
	return ret
}

func (c *Config) PrintEnv() {
	fmt.Println("---------------------")
	fmt.Printf("CHECK_INTERVAL=%d\n", c.CheckInterval)
	fmt.Printf("MAX_HISTORY_ENTRIES=%d\n", c.MaxHistoryEntries)
	fmt.Printf("CHECKS_FILE=%s\n", c.ChecksFile)
	fmt.Printf("INCIDENTS_FILE=%s\n", c.IncidentsFile)
	fmt.Printf("STATUS_HISTORY_FILE=%s\n", c.HistoryFile)
	fmt.Printf("PORT=%d\n", c.Port)
	fmt.Printf("TOKEN=%s\n", c.Token)
	fmt.Printf("CHATID=%s\n", c.Chatid)
	fmt.Println("---------------------")
}

func (c *Config) IndexHtmlFile() string {
	return "index.html"
}

func (c *Config) HistoryHtmlFile() string {
	return "history.html"
}

func (c *Config) ListenHost() string {
	return fmt.Sprintf(":%d", c.Port)
}

func (c *Config) ReadIncidentHtml() []byte {
	incidentMarkdown, err := os.ReadFile(c.IncidentsFile)
	if err != nil {
		log.Println("Failed to load incidents:", err)
		incidentMarkdown = []byte("## All Fine!")
	}
	return markdown.ToHTML(incidentMarkdown, nil, nil)
}

func (c *Config) ReadChecks() []Group {
	checksData, err := os.ReadFile(c.ChecksFile)
	if err != nil {
		log.Fatal("Failed to load checks file:", err)
	}
	var groups []Group
	if err = yaml.Unmarshal(checksData, &groups); err != nil {
		log.Fatal("Failed to parse checks file:", err)
	}
	return groups
}
