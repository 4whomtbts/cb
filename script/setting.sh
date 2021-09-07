#!/bin/bash

sudo useradd -s /bin/false -d /home/circuitbreaker -m sudo -G circuitbreaker \
    && echo 'circuitbreaker ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers


cat <<"EOF" > sudo tee -a /etc/systemd/system/circuitbreaker.service
[Unit]
Description=Circuitbreaker

[Service]
Type=simple
ExecStart=/home/circuitbreaker/bin/main
Restart=on-failure
EOF


