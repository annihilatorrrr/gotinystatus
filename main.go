package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Group struct {
	Title  string  `yaml:"title"`
	Checks []Check `yaml:"checks"`
}

type Check struct {
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	Host         string `yaml:"host"`
	Address      string `yaml:"address"`
	Port         int    `yaml:"Port"`
	ExpectedCode int    `yaml:"expected_code"`
}

type HistoryEntry struct {
	Timestamp string `json:"timestamp"`
	Status    bool   `json:"status"`
}

type GroupCheckResult struct {
	Title        string
	CheckResults []CheckResult
}

type CheckResult struct {
	Name   string
	Status bool
}

func checkHTTP(url string, expectedCode int) bool {
	client := &http.Client{Timeout: time.Second * 5}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == expectedCode
}

func pingIPv6(address string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("ping", "-n", "1", "-w", "5000", address)
	case "darwin":
		cmd = exec.Command("ping", "-c", "1", "-W", "5", address)
	default:
		cmd = exec.Command("ping", "-6", "-c", "1", "-W", "5", address)
	}
	return cmd.Run() == nil
}

func checkPing(host string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("ping", "-n", "1", "-w", "5000", host)
	default:
		cmd = exec.Command("ping", "-c", "1", "-W", "5", host)
	}
	return cmd.Run() == nil
}

func checkPort(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func runChecks(groups []Group) []GroupCheckResult {
	var results []GroupCheckResult
	groupResultsCh := make(chan GroupCheckResult, len(groups))
	for _, group := range groups {
		go func(g Group) {
			groupResultsCh <- checkGroup(g)
		}(group)
	}
	for i := 0; i < len(groups); i++ {
		results = append(results, <-groupResultsCh)
	}
	return results
}

func checkGroup(g Group) GroupCheckResult {
	var checkResults []CheckResult

	resultsCh := make(chan CheckResult, len(g.Checks))
	for _, check := range g.Checks {
		go func(c Check) {
			var status bool

			switch c.Type {
			case "http":
				status = checkHTTP(c.Host, c.ExpectedCode)
			case "ping":
				status = checkPing(c.Host)
			case "Port":
				status = checkPort(c.Host, c.Port)
			case "ipv6":
				status = pingIPv6(c.Address)
			}
			resultsCh <- CheckResult{c.Name, status}
		}(check)
	}
	for i := 0; i < len(g.Checks); i++ {
		checkResults = append(checkResults, <-resultsCh)
	}

	return GroupCheckResult{g.Title, checkResults}
}

func (c *Config) loadHistory() map[string][]HistoryEntry {
	file, err := os.Open(c.HistoryFile)
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

func (c *Config) saveHistory(history map[string][]HistoryEntry) {
	file, err := os.Create(c.HistoryFile)
	if err != nil {
		log.Println("Failed to save history:", err)
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	_ = json.NewEncoder(file).Encode(history)
}

func (c *Config) updateHistory(results []GroupCheckResult) {
	history := c.loadHistory()
	currentTime := time.Now().Format(time.RFC3339)
	for _, group := range results {
		for _, result := range group.CheckResults {
			name := result.Name
			if _, exists := history[name]; !exists {
				history[name] = []HistoryEntry{}
			}
			history[name] = append(history[name], HistoryEntry{currentTime, result.Status})
			sort.Slice(history[name], func(i, j int) bool {
				timeI, _ := time.Parse(time.RFC3339, history[name][i].Timestamp)
				timeJ, _ := time.Parse(time.RFC3339, history[name][j].Timestamp)
				return timeI.After(timeJ)
			})
			if len(history[name]) > c.MaxHistoryEntries {
				history[name] = history[name][:c.MaxHistoryEntries]
			}
		}
	}
	c.saveHistory(history)
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

func (c *Config) generateHistoryPage() {
	history := c.loadHistory()
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
	if err = os.WriteFile(c.HistoryHtmlFile(), buf.Bytes(), 0644); err != nil {
		log.Fatal("Failed to write history page:", err)
	}
}

func (c *Config) monitorServices() {
	for {
		groups := c.ReadChecks()
		//log.Printf("Groups: %+v", groups)
		results := runChecks(groups)
		c.updateHistory(results)
		data := map[string]interface{}{
			"groups":       results,
			"incidents":    template.HTML(c.ReadIncidentHtml()),
			"last_updated": time.Now().Format("2006-01-02 15:04:05"),
		}
		html := renderTemplate(data)
		if err := os.WriteFile(c.IndexHtmlFile(), []byte(html), 0644); err != nil {
			log.Fatal("Failed to write index.html:", err)
		}
		c.generateHistoryPage()
		log.Println("Status pages updated!")
		if c.Token != "" && c.Chatid != "" {
			log.Println("Notifying on telegram ...")
			for key, data := range c.loadHistory() {
				if total := len(data); total >= 2 {
					latestdata := data[:2]
					if latestdata[0].Status == latestdata[1].Status {
						continue
					}
					lastst := latestdata[1].Status
					newinterval := c.CheckInterval
					for x, y := range data {
						if x > 1 {
							if y.Status == lastst {
								newinterval += 60
							} else {
								break
							}
						}
					}
					tosend := fmt.Sprintf("<b>✅ %s is now Up!</b>\nSeen Down from last %ds!", key, newinterval)
					if !latestdata[0].Status {
						tosend = fmt.Sprintf("<b> 🛑 %s is now Down!</b>\nWas seen Up from last %ds!", key, newinterval)
					}
					_ = checkHTTP(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&parse_mode=html&text=%s", c.Token, c.Chatid, url.QueryEscape(tosend)), 200)
				} else {
					tosend := fmt.Sprintf("<b> 🛑 %s is now Down!</b>", key)
					if data[0].Status {
						tosend = fmt.Sprintf("<b>✅ %s is now Up!</b>", key)
					}
					_ = checkHTTP(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&parse_mode=html&text=%s", c.Token, c.Chatid, url.QueryEscape(tosend)), 200)
				}
			}
			log.Println("Notified on telegram!")
		}
		time.Sleep(time.Duration(c.CheckInterval) * time.Second)
	}
}

func serveFile(w http.ResponseWriter, r *http.Request, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filePath)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed!", http.StatusMethodNotAllowed)
		return
	}
	_, _ = fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Service Status</title>
    <style>
        body {
            background-color: #121212;
            color: #e0e0e0;
            font-family: Arial, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
        }
        h1 {
            color: #bb86fc;
        }
        pre {
            background-color: #1e1e1e;
            padding: 10px;
            border-radius: 5px;
            overflow-x: auto;
        }
    </style>
</head>
<body>
    <h1>I'm alive!</h1>
    <p>Go Version: %s</p>
    <p>Go Routines: %d</p>
    <p>Source Code: <a href="https://github.com/annihilatorrrr/gotinystatus" style="color: #bb86fc;">Gotinystatus</a></p>
</body>
</html>`, runtime.Version(), runtime.NumGoroutine())
}

func main() {
	c := readEnv()
	log.Println("Monitoring services ...")

	c.PrintEnv()

	if c.Port != 0 {
		go c.monitorServices()
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				serveFile(w, r, "./"+c.IndexHtmlFile())
			} else {
				http.NotFound(w, r)
			}
		})
		http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/status") {
				handleHome(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
		http.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/history") {
				serveFile(w, r, "./"+c.HistoryHtmlFile())
			} else {
				http.NotFound(w, r)
			}
		})
		log.Println("Server started!")
		if err := http.ListenAndServe(c.ListenHost(), nil); err != nil {
			log.Println(err.Error())
		}
	} else {
		c.monitorServices()
	}
	log.Println("Bye!")
}
