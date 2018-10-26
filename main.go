package main

import (
	"net/http"

	"github.com/Cryptodog/bex-attachment-server/server"
	"github.com/superp00t/etc/yo"
)

func sserver(s []string) {
	yo.Fatal(http.ListenAndServe(yo.StringG("l"), server.New(yo.StringG("d"), yo.Int64G("s"))))
}

func main() {
	yo.Int64f("s", "storage-limit", "the maxmimum number of bytes you can store", 300*server.MB)
	yo.Stringf("d", "data-location", "the location where files are stored", "/tmp/bex-attachments/")
	yo.Stringf("l", "listen", "the IP address:port to listen on", ":8050")
	yo.Boolf("p", "proxied", "use X-Real-IP header from NGINX")
	yo.Main("allows users to upload files using the Cryptodog BEX API", sserver)
	yo.Init()
}
