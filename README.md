# BEX Attachment Server

This server implements [the BEX Attachment Server specification](https://github.com/Cryptodog/cryptodog/wiki/Binary-Extensions-(BEX)-Draft).

It also exposes a [TURN server](https://github.com/pions/turn) in order to avoid IP leaks caused by BEX's WebRTC extensions.

# Goals

- Allow high request volume
- Erase old files in the event of high load
- Block spammers without being overzealous
- Provide users with reasons their uploads have failed

# Setup

```bash
# install Go and git if you haven't already
sudo apt-get install git golang

go get -u -v github.com/Cryptodog/bex-attachment-server

~/go/bin/bex-attachment-server --config ~/bex.toml
```

If you want to use it behind an NGINX proxy with the X-Real-IP header, use the flag -p, --proxied