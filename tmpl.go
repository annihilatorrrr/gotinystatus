package main

const templateFile = `<!DOCTYPE html>
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
{{range .groups}}
<h3>{{.Title}} Status</h3>
<div class="status-grid">
    {{range .CheckResults}}
    <div class="status-item">
        <h3>{{.Name}}</h3>
        <p class="{{if .Status}}status-up{{else}}status-down{{end}}">
            {{if .Status}}Operational{{else}}Down{{end}}
        </p>
    </div>
    {{end}}
</div>
{{end}}
<h2>Incident History</h2>
<div class="incidents">
    {{.incidents}}
</div>
<div class="footer">
    <p>Last updated: {{.last_updated}}</p>
    <p><a href="history">View Status History</a></p>
	<p>Powered by <a href="https://github.com/annihilatorrrr/gotinystatus">GoTinyStatus</a></p>
</div>
</body>
</html>`

const historyTemplateFile = `<!DOCTYPE html>
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
    <p><a href="/">Back to Current Status</a></p>
	<p>Powered by <a href="https://github.com/annihilatorrrr/gotinystatus">GoTinyStatus</a></p>
</div>
</body>
</html>`
