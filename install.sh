#!bin/bash

OS=linux
if [ "$(uname)" == "Darwin" ]; then
	OS="darwin"
fi

curl -s -L -o /usr/local/bin/fusera https://github.com/mitre/fusera/releases/download/v0.0.8/fusera-${OS}-amd64
curl -s -L -o /usr/local/bin/sracp https://github.com/mitre/fusera/releases/download/v0.0.8/sracp-${OS}-amd64

chmod +x /usr/local/bin/fusera
chmod +x /usr/local/bin/sracp
