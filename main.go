package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Check struct {
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	ExpectedCode int    `yaml:"expected_code"`
}

type HistoryEntry struct {
	Timestamp string `json:"timestamp"`
	Status    bool   `json:"status"`
}

var (
	checkInterval       = getEnvInt("CHECK_INTERVAL", 30)
	maxHistoryEntries   = getEnvInt("MAX_HISTORY_ENTRIES", 10)
	checksFile          = getEnv("CHECKS_FILE", "checks.yaml")
	incidentsFile       = getEnv("INCIDENTS_FILE", "incidents.html")
	templateFile        = getEnv("TEMPLATE_FILE", "index.template.html")
	historyTemplateFile = getEnv("HISTORY_TEMPLATE_FILE", "history.template.html")
	historyFile         = getEnv("STATUS_HISTORY_FILE", "history.json")
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		var intValue int
		_, _ = fmt.Sscanf(value, "%d", &intValue)
		return intValue
	}
	return fallback
}

func checkHTTP(url string, expectedCode int) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	return resp.StatusCode == expectedCode
}

func checkPing(host string) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "2", host)
	return cmd.Run() == nil
}

func checkPort(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 2*time.Second)
	if err != nil {
		return false
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	return true
}

func runChecks(checks []Check) []map[string]interface{} {
	var results []map[string]interface{}
	for _, check := range checks {
		var status bool
		switch check.Type {
		case "http":
			status = checkHTTP(check.Host, check.ExpectedCode)
		case "ping":
			status = checkPing(check.Host)
		case "port":
			status = checkPort(check.Host, check.Port)
		}
		results = append(results, map[string]interface{}{"name": check.Name, "status": status})
	}
	return results
}

func loadHistory() map[string][]HistoryEntry {
	file, err := os.Open(historyFile)
	if err != nil {
		return map[string][]HistoryEntry{}
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	var history map[string][]HistoryEntry
	_ = json.NewDecoder(file).Decode(&history)
	return history
}

func saveHistory(history map[string][]HistoryEntry) {
	file, err := os.Create(historyFile)
	if err != nil {
		log.Println("Failed to save history:", err)
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	_ = json.NewEncoder(file).Encode(history)
}

func updateHistory(results []map[string]interface{}) {
	history := loadHistory()
	currentTime := time.Now().Format(time.RFC3339)
	for _, result := range results {
		name := result["name"].(string)
		if _, exists := history[name]; !exists {
			history[name] = []HistoryEntry{}
		}
		history[name] = append(history[name], HistoryEntry{currentTime, result["status"].(bool)})
		if len(history[name]) > maxHistoryEntries {
			history[name] = history[name][len(history[name])-maxHistoryEntries:]
		}
	}
	saveHistory(history)
}

func renderTemplate(templateFile string, data map[string]interface{}) string {
	tmplBytes, err := os.ReadFile(templateFile)
	if err != nil {
		log.Fatal(err)
	}
	funcMap := template.FuncMap{
		"check": func(status bool) string {
			if status {
				return "Up"
			}
			return "Down"
		},
	}
	tmpl, err := template.New("status").Funcs(funcMap).Parse(string(tmplBytes))
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		log.Fatal(err)
	}
	return buf.String()
}

func generateHistoryPage() {
	history := loadHistory()
	tmplBytes, err := os.ReadFile(historyTemplateFile)
	if err != nil {
		log.Fatal("Failed to read history template:", err)
	}
	tmpl, err := template.New("history").Funcs(template.FuncMap{
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
	}).Parse(string(tmplBytes))
	if err != nil {
		log.Fatal("Failed to parse history template:", err)
	}
	data := map[string]interface{}{
		"history":      history,
		"last_updated": time.Now().Format("2006-01-02 15:04:05"),
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		log.Fatal("Failed to execute history template:", err)
	}
	if err = os.WriteFile("history.html", buf.Bytes(), 0644); err != nil {
		log.Fatal("Failed to write history page:", err)
	}
}

func monitorServices() {
	for {
		checksData, err := os.ReadFile(checksFile)
		if err != nil {
			log.Fatal("Failed to load checks file:", err)
		}
		var checks []Check
		if err = yaml.Unmarshal(checksData, &checks); err != nil {
			log.Fatal("Failed to parse checks file:", err)
		}
		results := runChecks(checks)
		updateHistory(results)
		incidentMarkdown, err := os.ReadFile(incidentsFile)
		if err != nil {
			log.Println("Failed to load incidents:", err)
			incidentMarkdown = []byte("<h2>All Fine!</h2>")
		}
		data := map[string]interface{}{
			"checks":       results,
			"incidents":    template.HTML(incidentMarkdown),
			"last_updated": time.Now().Format("2006-01-02 15:04:05"),
		}
		html := renderTemplate(templateFile, data)
		if err = os.WriteFile("index.html", []byte(html), 0644); err != nil {
			log.Fatal("Failed to write index.html:", err)
		}
		generateHistoryPage()
		log.Println("Status pages updated!")
		time.Sleep(time.Duration(checkInterval) * time.Second)
	}
}

func main() {
	log.Println("Monitoring services ...")
	monitorServices()
}
