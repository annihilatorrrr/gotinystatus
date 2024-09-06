package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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

const (
	templateFile = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go TinyStatus</title>
    <style>
        body {
            font-family: sans-serif;
            line-height: 1.6;
            color: #e0e0e0;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #181818;
            transition: background 0.3s ease, color 0.3s ease;
        }
        h1, h2 {
            color: #e0e0e0;
            text-align: center;
        }
        .status-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 40px;
        }
        .status-item {
            background: #242424;
            border-radius: 8px;
            padding: 15px;
            box-shadow: 0 2px 4px rgba(255,255,255,0.1);
            text-align: center;
            transition: transform .2s, background 0.3s ease;
        }
        .status-item:hover {
            transform: translateY(-5px);
        }
        .status-item h3 {
            margin: 0 0 10px;
        }
        .status-up { color: #27ae60; }
        .status-down { color: #e74c3c; }
        .incidents {
            background: #242424;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 2px 4px rgba(255,255,255,0.1);
            margin-bottom: 40px;
        }
        .footer {
            text-align: center;
            font-size: .9em;
            color: #a0a0a0;
            margin-top: 40px;
        }
        .footer a {
            color: #9b59b6;
            text-decoration: none;
        }
        .footer a:hover { text-decoration: underline; }
    </style>
</head>
<body>
<h1>Go TinyStatus</h1>
<h2>Current Status:</h2>
<div class="status-grid">
    {{range .checks}}
    <div class="status-item">
        <h3>{{.name}}</h3>
        <p class="{{if .status}}status-up{{else}}status-down{{end}}">
            {{if .status}}Operational{{else}}Down{{end}}
        </p>
    </div>
    {{end}}
</div>
<h2>Incident History</h2>
<div class="incidents">
    {{.incidents}}
</div>
<div class="footer">
    <p>Last updated: {{.last_updated}}</p>
    <p>Powered by <a href="https://github.com/annihilatorrrr/gotinystatus">GoTinyStatus</a></p>
    <p><a href="history">View Status History</a></p>
</div>
</body>
</html>`
	historyTemplateFile = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go TinyStatus History</title>
    <style>
        body {
            font-family: sans-serif;
            line-height: 1.6;
            color: #e0e0e0;
            max-width: 1200px;
            margin: auto;
            padding: 20px;
            background: #181818;
            transition: background 0.3s ease, color 0.3s ease;
        }
        h1, h2 {
            color: #e0e0e0;
            text-align: center;
        }
        .history-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .history-item {
            background: #242424;
            border-radius: 8px;
            padding: 15px;
            box-shadow: 0 2px 4px rgba(255,255,255,0.1);
            max-height: 300px;
            overflow: auto;
        }
        .history-item h2 {
            font-size: 1.2rem;
            margin: 0;
        }
        .history-entry {
            margin-bottom: 5px;
            font-size: 0.9rem;
            display: flex;
            justify-content: space-between;
        }
        .status-up { color: #27ae60; }
        .status-down { color: #e74c3c; }
        .footer {
            text-align: center;
            font-size: .9em;
            color: #a0a0a0;
            margin-top: 40px;
        }
        .footer a {
            color: #9b59b6;
            text-decoration: none;
        }
        .footer a:hover { text-decoration: underline; }
    </style>
</head>
<body>
<h1>Go TinyStatus History</h1>
<div class="history-grid">
    {{ range $service, $entries := .history }}
    <div class="history-item">
        <h2>{{ $service }}</h2>
        {{ range $entry := $entries }}
        <div class="history-entry">
            <span>{{ index (split $entry.Timestamp "T") 0 }} {{ slice (index (split $entry.Timestamp "T") 1) 0 8 }}</span>
            <span class="{{ if $entry.Status }}status-up{{ else }}status-down{{ end }}">
                {{ if $entry.Status }}Up{{ else }}Down{{ end }}
            </span>
        </div>
        {{ end }}
    </div>
    {{ end }}
</div>
<div class="footer">
    <p>Last updated: {{.last_updated}}</p>
    <p>Powered by <a href="https://github.com/annihilatorrrr/gotinystatus">GoTinyStatus</a></p>
    <p><a href="/">Back to Current Status</a></p>
</div>
</body>
</html>`
	indexfile   = "index.html"
	historyfile = "history.html"
)

var (
	checkInterval     = getEnvInt("CHECK_INTERVAL", 30)
	maxHistoryEntries = getEnvInt("MAX_HISTORY_ENTRIES", 10)
	checksFile        = getEnv("CHECKS_FILE", "checks.yaml")
	incidentsFile     = getEnv("INCIDENTS_FILE", "incidents.html")
	historyFile       = getEnv("STATUS_HISTORY_FILE", "history.json")
	port 		  = getEnv("PORT", "")
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
	resultsCh := make(chan map[string]interface{}, len(checks))
	for _, check := range checks {
		go func(c Check) {
			var status bool
			switch c.Type {
			case "http":
				status = checkHTTP(c.Host, c.ExpectedCode)
			case "ping":
				status = checkPing(c.Host)
			case "port":
				status = checkPort(c.Host, c.Port)
			}
			resultsCh <- map[string]interface{}{"name": c.Name, "status": status}
		}(check)
	}
	for i := 0; i < len(checks); i++ {
		results = append(results, <-resultsCh)
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
	if history == nil {
		history = make(map[string][]HistoryEntry)
	}
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

func renderTemplate(data map[string]interface{}) string {
	funcMap := template.FuncMap{
		"check": func(status bool) string {
			if status {
				return "Up"
			}
			return "Down"
		},
	}
	tmpl, err := template.New("status").Funcs(funcMap).Parse(templateFile)
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
	tmpl, err := template.New("history").Funcs(template.FuncMap{
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
	}).Parse(historyTemplateFile)
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
	if err = os.WriteFile(historyfile, buf.Bytes(), 0644); err != nil {
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
		html := renderTemplate(data)
		if err = os.WriteFile(indexfile, []byte(html), 0644); err != nil {
			log.Fatal("Failed to write index.html:", err)
		}
		generateHistoryPage()
		log.Println("Status pages updated!")
		time.Sleep(time.Duration(checkInterval) * time.Second)
	}
}

func serveFile(w http.ResponseWriter, r *http.Request, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filePath)
}

func main() {
	log.Println("Monitoring services ...")
	if port != "" {
		go monitorServices()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				serveFile(w, r, "./"+indexfile)
			} else {
				http.NotFound(w, r)
			}
		})
		http.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/history" {
				serveFile(w, r, "./"+historyfile)
			} else {
				http.NotFound(w, r)
			}
		})
		log.Println("Server started!")
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal(err)
		}
	} else {
		monitorServices()
	}
}
