#!/bin/bash 

mkdir -p ../certs
openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 -nodes -out ../certs/proxybatcher.crt -keyout ../certs/proxybatcher.key
 

