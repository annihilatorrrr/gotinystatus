# GoTinyStatus

GoTinyStatus is a simple, customizable status page generator that allows you to monitor the status of various services and display them on a clean, dark mode, responsive web page. [Check out an online demo.](https://status.memersgallery.tech/)


![Demo](https://github.com/user-attachments/assets/9611f924-22c5-4335-ab78-84b771edc023)

## NOTE:

- Same as https://github.com/harsxv/tinystatus BUT made in GO!

## Features

- Monitor HTTP endpoints, ping hosts, ipv6 address and check open ports
- Responsive design for both status page and history page
- Customizable service checks via YAML configuration
- Incident history tracking
- Automatic status updates at configurable intervals
- The generated HTML is only 5KB in size
- Telegram notification.

## Prerequisites

- Go

## Installation

1. Clone the repository or download the source code:
   ```
   git clone https://github.com/annihilatorrrr/gotinystatus.git
   cd gotinystatus
   ```

2. Install the required dependencies and run:
   ```
   go run .
   ```

## Configuration

1. Create a `.env` file in the project root and customize the variables:
   ```
   CHECK_INTERVAL=30
   MAX_HISTORY_ENTRIES=10
   CHECKS_FILE=checks.yaml
   INCIDENTS_FILE=incidents.html
   STATUS_HISTORY_FILE=history.json
   PORT= Optional Port
   TOKEN= Optional Telegram Bot TOKEN
   CHATID= Optional Telegram Chat Id for notification
   ```

2. Edit the `checks.yaml` file to add or modify the services you want to monitor. Example:
   ```yaml
   - name: GitHub Home
     type: http
     host: https://github.com
     expected_code: 200

   - name: Google DNS
     type: ping
     host: 8.8.8.8

   - name: Database
     type: port
     host: db.example.com
     port: 5432
   ```

3. (Optional) Customize the `incidents.md` file to add any known incidents or maintenance schedules.

4. (Optional) Modify the `templateFile` and `historyTemplateFile` constant to customize the look and feel of your status pages.

## Usage

1. Run the TinyStatus script:
   ```
   go run main.go
   ```

2. The script will generate 3 files:
   - `index.html`: The main status page
   - `history.html`: The status history page
   - `history.json`: The status history and timestamp hdata

3. To keep the status page continuously updated, you can run the script in the background:
   - On Unix-like systems (Linux, macOS):
   Build:
     ```
     go build -ldflags="-w -s" .
     ```
   Now just run the go app as service.
   - On Windows, you can use the Task Scheduler to run the exe file at startup.

4. Serve the generated HTML files using HTTP server at specific PORT.

## Using Docker

In order to run the script using Docker:

   ```
    docker build -t gotinystatus .
    docker run -ti --rm --name gotinystatus -v "$PWD":/usr/src/myapp -w /usr/src/myapp gotinystatus
   ```

## Customization

- Adjust the configuration variables in the `.env` file to customize the behavior of GoTinyStatus.
- Customize the appearance of the status page by editing the CSS in `templateFile` and `historyTemplateFile`.
- Add or remove services by modifying the `checks.yaml` file.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is open source and available under the [MIT License](LICENSE).
