# uptime 

   A simple, reliable uptime monitoring service written in Go.

## 📖 Overview

uptime is a lightweight HTTP service that periodically checks your configured URLs or services and alerts when they go down. Designed for simplicity and flexibility, it supports customizable monitoring intervals, multiple notification channels.
### Features

  * Monitor HTTP(s) endpoints (TCP, and ICMP (ping) endpoints coming soon)

  *  Per-endpoint settings: interval, timeout, retries

  * Notification methods: Email(mailgun), Slack, Discord

  *  Local dashboard and basic API

  *  Lightweight and reliable — single binary, no dependencies


## 🛠️ Installation
From Source
```
git clone https://github.com/devilcove/uptime.git
cd uptime/helper
# run helper program to creaate initial admin user
go run .
cd ..
go build
./uptime
```
Then log in with the previous created user to the dashboard at http://localhost:8090 to view status, logs, and metrics.   

Systemd  
example service file in files/uptime.service

## 🚀 Usage

Supported Endpoint Types

* http: standard HTTP health check (status, response time, certificate expiry)

* tcp: connect to host:port (coming soon)

* ping: ICMP echo tests (coming soon)

## 🧩 Notifications

Configure how you're notified on failures—support includes:

* Email: via Mailgun

* Slack: via slack app

* Discord: via webhook

Choose one or combine multiple.

## Screenshots
![status page](/files/screenshots/status.png)  
![details page](/files/screenshots/details.png)  
![history](/files/screenshots/history.png)  
![notifications](/files/screenshots/notifications.png)  
![new monitor](/files/screenshots/new_monitor.png)  
![new notification](/files/screenshots/new_notification.png)  
