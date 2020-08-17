#!/bin/bash
set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"


# install go

sudo apt-get -y update
sudo apt-get install -y golang

command -v go >/dev/null 2>&1 || { 
   echo >&2 "go does not appear to have installed correctly, exiting..."; exit 1; 
}

#compile the program

echo "Entering $DIR/proxybatcher and compiling"
cd $DIR/proxybatcher
go build
chmod 755 proxybatcher

echo "Renaming and moving binary to $DIR/proxybatcher/sonarproxybatcher"

mv "$DIR/proxybatcher/proxybatcher" "sonarproxybatcher"
cp "$DIR/proxybatcher/sonarproxybatcher" "$DIR"

nano $DIR/conf/proxybatcher.yaml


