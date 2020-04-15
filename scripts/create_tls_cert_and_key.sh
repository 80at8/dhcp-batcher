#!/bin/bash 

mkdir -p /opt/sonar/dhcp-batcher/tls/
openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out /opt/sonar/dhcp-batcher/tls/dhcp-batcher.crt -keyout /opt/sonar/dhcp-batcher/tls/dhcp-batcher.key
 

