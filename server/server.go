package server

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/pions/pkg/stun"
	"github.com/pions/turn"
	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"
)

// 32 megabytes
var megabyte int64 = 1024 * 1024
var maximumBytes int64 = 64 * megabyte
var maxUploadSize int64 = 12 * megabyte

type server struct {
	// the maximum number of files to be stored
	storageLimit int64
	usedBytes    int64
	location     string
	IPData       *sync.Map
	l            *sync.Mutex
	realm        string
	proxied      bool

	udpPort int

	accs map[string]string

	router http.Handler
}

type spam struct {
	IP            string
	bytesUploaded int64
	upload        chan bool
	l             *sync.Mutex
}

func (s *server) addUsedBytes(i int64) {
	s.l.Lock()
	s.usedBytes += i
	s.l.Unlock()
}

func (s *server) availableSpace() int64 {
	du := DiskUsage(s.location)
	free := int64(du.Free)

	if s.storageLimit < free {
		return s.storageLimit
	}

	if free < s.storageLimit {
		return free
	}

	return free
}

func (s *server) AuthenticateRequest(username string, srcAddr *stun.TransportAddr) (password string, ok bool) {
	if password, ok := s.accs[username]; ok {
		return password, true
	}
	return "", false
}

func New(c *Config) http.Handler {
	s := new(server)
	s.IPData = new(sync.Map)
	s.location = c.DataLocation
	s.storageLimit = c.StorageLimit
	s.l = new(sync.Mutex)
	p := etc.ParseSystemPath(s.location)
	if !p.IsExtant() {
		os.MkdirAll(s.location, 0700)
	}

	if c.TURNAddress != "" {
		s.accs = c.Accounts
		s.realm = c.Realm
		ad, err := net.ResolveUDPAddr("udp", c.TURNAddress)
		if err != nil {
			yo.Fatal(err)
		}

		s.udpPort = ad.Port

		go func() {
			turn.Start(turn.StartArguments{
				Server:  s,
				Realm:   s.realm,
				UDPPort: s.udpPort,
			})
		}()
	}

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/upload", s.checkSpam(s.upload)).Methods("POST")
	r.HandleFunc("/statistics.json", s.statistics).Methods("GET")
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(s.location))))

	s.router = r
	return s
}

func (s *server) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Access-Control-Allow-Methods", "*")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Header().Set("Access-Control-Allow-Headers", "*")
	rw.Header().Set("Access-Control-Max-Age", "3600")

	s.router.ServeHTTP(rw, r)
}

func (s *server) checkSpam(fn func(rw http.ResponseWriter, r *http.Request, s *spam)) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var ip string
		if s.proxied {
			ip = r.Header.Get("X-Real-IP")
		} else {
			i, err := net.ResolveTCPAddr("tcp", r.RemoteAddr)
			if err != nil {
				rw.WriteHeader(http.StatusBadGateway)
				return
			}

			ip = i.IP.String()
		}

		var id *spam
		d, ok := s.IPData.Load(ip)
		if !ok {
			id = new(spam)
			id.l = new(sync.Mutex)
			id.IP = ip
			s.IPData.Store(ip, id)
			go s.watchIP(id)
		} else {
			id = d.(*spam)
		}

		id.l.Lock()
		if id.bytesUploaded > maximumBytes {
			rw.WriteHeader(http.StatusTooManyRequests)
			id.l.Unlock()
			return
		}
		id.l.Unlock()

		fn(rw, r, id)
	}
}

func (s *server) watchIP(id *spam) {
	timeout := 10 * time.Minute
	for {
		select {
		case <-time.After(timeout):
			s.IPData.Delete(id.IP)
			return
		case <-id.upload:
			timeout += 3 * time.Minute
		}
	}
}

func (s *server) upload(rw http.ResponseWriter, r *http.Request, id *spam) {
	if r.Method != "POST" {
		return
	}

	p := etc.ParseSystemPath(s.location)

	_cl := r.URL.Query().Get("cl")
	cl, err := strconv.ParseInt(_cl, 0, 64)
	if err != nil {
		yo.Warn(err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	yo.Println("Uploading", _cl, id.IP)

	if cl > maxUploadSize {
		rw.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	// this is a problem. we need to free up space quickly.
	if s.availableSpace() < cl {
		ir, err := ioutil.ReadDir(s.location)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Find the oldest file.
		index := -1
		lastTime := time.Second * 0

		for i, v := range ir {
			t := time.Now()

			if t.Sub(v.ModTime()) > lastTime {
				lastTime = t.Sub(v.ModTime())
				index = i
			}
		}

		if index == -1 {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if lastTime < 3*time.Minute {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		} else {
			sz := ir[index].Size()
			path := p.Concat(ir[index].Name()).Render()
			os.Remove(path)
			s.addUsedBytes(-1 * sz)
		}

	}

	var u etc.UUID

	for {
		u = etc.GenerateRandomUUID()
		pth := p.Concat(u.String()).IsExtant()
		if pth {
			continue
		} else {
			break
		}
	}

	s.addUsedBytes(cl)

	f, err := etc.FileController(p.Concat(u.String()).Render())
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rd := &io.LimitedReader{
		r.Body,
		cl,
	}

	wr, err := io.Copy(f, rd)
	if err != nil {
		yo.Warn(err)
	}

	f.Close()

	if wr != cl {
		yo.Println("Expected", cl, "got", wr)
		f.Delete()

		rw.WriteHeader(http.StatusConflict)
		return
	}

	e := etc.NewBuffer()
	e.WriteUUID(u)
	rw.Write(e.Bytes())
	id.l.Lock()
	id.bytesUploaded += cl
	id.l.Unlock()

	s.addUsedBytes(cl)

	go func() {
		id.upload <- true
	}()

	go func() {
		time.Sleep(10 * time.Minute)
		if p.Concat(u.String()).IsExtant() {
			os.Remove(p.Concat(u.String()).Render())
			s.addUsedBytes(-1 * cl)
		}

		id.l.Lock()
		id.bytesUploaded -= cl
		id.l.Unlock()
	}()
}

func (s *server) statistics(rw http.ResponseWriter, r *http.Request) {
	j := json.NewEncoder(rw)
	j.SetIndent("", "  ")
	j.Encode(struct {
		UsedBytes      int64 `json:"used_bytes"`
		Limit          int64 `json:"limit"`
		AvailableSpace int64 `json:"available_space"`
	}{
		s.usedBytes,
		s.storageLimit,
		s.availableSpace(),
	})
}
