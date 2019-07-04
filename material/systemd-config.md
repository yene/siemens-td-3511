# Systemd Config

Run the binary every 15 minutes. This requires a service and a timer which has to be enabled and started.

Setup systemd, a service for the timer:
`sudo nano /etc/systemd/system/readvalues.service`
(check that your binary path in ExecStart is correct)

> The service does not have to be enabled, the timer will start it.

```
[Unit]
Description=Read Energy Values
After=network.target

[Service]
User=pi
ExecStart=/home/pi/read-values

[Install]
WantedBy=multi-user.target
```

Setup systemd, timer for the service:
`sudo nano /etc/systemd/system/readvalues.timer`

```
[Unit]
Description=Run readvalues

[Timer]
OnCalendar=*:0/5
Unit=readvalues.service

[Install]
WantedBy=timers.target
```

`sudo systemctl daemon-reload`
`sudo systemctl enable readvalues.timer`
`sudo systemctl start readvalues.timer`

Get timer status with: `systemctl list-timers --all`
