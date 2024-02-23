enable CGO
go build main.go

create service
sudo nano /lib/systemd/system/go_app.service

[Unit]
Description=Example golang
[Service]
Type=simple
Restart=always
RestartSec=5s
WorkingDirectory=/home/ubuntu/image-resize/
ExecStart=/home/ubuntu/image-resize/main
[Install]
WantedBy=multi-user.target

sudo chmod +x /home/ubuntu/image-resize/main

sudo systemctl enable go_app
sudo systemctl start go_app
sudo systemctl status go_app

set reverse proxy nginx
https://www.digitalocean.com/community/tutorials/how-to-configure-nginx-as-a-reverse-proxy-on-ubuntu-22-04
set ssl
https://www.digitalocean.com/community/tutorials/how-to-secure-nginx-with-let-s-encrypt-on-ubuntu-22-04