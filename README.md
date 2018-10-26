# BEX Attachment Server

This server is intended to provide a host for a simple and resilient encrypted file transfer API for Cryptodog servers to replace the slow, unstable and outdated transfer method which uses the (now unsupported) SI filetransfer mechanism with encrypted In-Band Bytestreams.

# Goals

- Allow high request volume
- Erase old files in the event of high load
- Block spammers without being overzealous
- Provide users with reasons their uploads have failed

# Setup

```bash
# install Go if you haven't already
sudo apt-get install git golang

go get -u -v github.com/Cryptodog/bex-attachment-server

mkdir /tmp/bex-attachments
~/go/bin/bex-attachment-server --listen <ip address> --data-location=/tmp/bex-attachments
```

If you want to use it behind an NGINX proxy with the X-Real-IP header, use the flag -p, --proxied