package main

import (
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/superp00t/etc"

	"github.com/Cryptodog/bex-attachment-server/server"
	"github.com/superp00t/etc/yo"
)

func getConfig(path string) *server.Config {
	if etc.ParseSystemPath(path).IsExtant() == false {
		fl, err := etc.FileController(path)
		if err != nil {
			yo.Fatal(err)
		}

		enc := toml.NewEncoder(fl)
		err = enc.Encode(&server.Config{
			AttachmentAddress: ":8080",
			TURNAddress:       "0.0.0.0:3478",
			StorageLimit:      300 * server.MB,
			DataLocation:      "/tmp/bex-attachments",
			Proxied:           false,
			Accounts: map[string]string{
				"user": "password",
			},
		})
		if err != nil {
			yo.Fatal(err)
		}

		yo.Ok("A basic configuration file has been created at", path, ". You should edit it to your liking.")
		os.Exit(0)
	}

	f, err := etc.FileController(path)
	if err != nil {
		yo.Fatal(err)
	}

	cfg := new(server.Config)
	_, err = toml.DecodeReader(f, cfg)

	if err != nil {
		yo.Fatal(err)
	}

	f.Close()

	return cfg
}

func sserver(s []string) {
	cerr := make(chan error)

	cfg := getConfig(yo.StringG("c"))

	go func() {
		cerr <- http.ListenAndServe(cfg.AttachmentAddress, server.New(cfg))
	}()

	yo.Fatal(<-cerr)
}

func main() {
	yo.Stringf("c", "config", "configuration file", os.Getenv("HOME")+"/config.toml")
	yo.Main("allows users to upload files using the Cryptodog BEX API", sserver)
	yo.Init()
}
