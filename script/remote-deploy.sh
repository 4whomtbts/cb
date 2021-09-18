#!/bin/bash

SUDOER=$1
HOME=/home/$SUDOER
CB_HOME=/home/circuitbreaker
CB_BIN=circuitbreaker-bin

sudo useradd -s /bin/false -d /$CB_HOME -m -G sudo circuitbreaker \
    && echo 'circuitbreaker ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers
sudo mkdir -p $CB_HOME/bin /etc/cb
sudo cp $HOME/$CB_BIN $CB_HOME/bin
sudo chown -R circuitbreaker: $CB_HOME

sudo systemctl stop circuitbreaker
SERVICE_FILE=/etc/systemd/system/circuitbreaker.service
sudo rm -rf $SERVICE_FILE
cat <<EOF | sudo tee $SERVICE_FILE
[Unit]
Description=Circuitbreaker

[Service]
Type=simple
ExecStart=$CB_HOME/bin/$CB_BIN
Restart=on-failure
EOF

sudo systemctl daemon-reload
sudo systemctl start circuitbreaker