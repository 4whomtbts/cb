#!/bin/bash

IP=$1
PORT=$2
NODE_NAME=$3
SUDOER=$4
TYPE=${5:-'node'}
SUDOER_HOME=/home/$SUDOER

scp -P $PORT ./remote-deploy.sh $SUDOER@$IP:$SUDOER_HOME
cat <<EOF | sudo tee circuitbreaker.yaml
nodeName: $1
type: node
port: 5000

exporters:
  - name: node_exporter
    label: node_exporter
    url: http://localhost:9100/metrics
    config:
      maxCpuTemp: 90
  - name: dcgm_exporter
    label: dcgm_exporter
    url: http://localhost:9400/metrics
    config:
      maxGpuTemp: 90
EOF

cat <<EOF | sudo tee circuitbreaker-master.yaml
nodeName: $NODE_NAME
type: master
port: 5000

maxTemperature: 25
healthCheckIntervalSec: 43200
circuitBreakerIntervalSec: 300
email: ailabcircuitbreaker.dgu@gmail.com
emailPassword: ***
emailReceivers:
  - 4whomtbts@gmail.com

nodes:
  - http://192.168.1.11:5000
  - http://192.168.1.12:5000
  - http://192.168.1.13:5000
  - http://192.168.1.14:5000
  - http://192.168.1.15:5000
  - http://192.168.1.16:5000
  - http://192.168.1.17:5000

exporters:
  - name: node_exporter
    label: node_exporter
    url: http://localhost:9100/metrics
    config:
      maxCpuTemp: 90
  - name: dcgm_exporter
    label: dcgm_exporter
    url: http://localhost:9400/metrics
    config:
      maxGpuTemp: 90
EOF

if [ $TYPE == "master" ]; then
  scp -P $PORT ./circuitbreaker-master.yaml $SUDOER@$IP:$SUDOER_HOME/circuitbreaker.yaml
else
  scp -P $PORT ./circuitbreaker.yaml $SUDOER@$IP:$SUDOER_HOME
fi
scp -P $PORT ../script/bin/circuitbreaker $SUDOER@$IP:$SUDOER_HOME/circuitbreaker-bin
ssh -t -p $PORT $SUDOER@$IP 'sudo sh ./remote-deploy.sh'" $SUDOER"


