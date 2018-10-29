package server

type Config struct {
	AttachmentAddress string
	TURNAddress       string
	StorageLimit      int64
	DataLocation      string
	Proxied           bool
	Realm             string
	Accounts          map[string]string
}
