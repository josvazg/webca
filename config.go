package webca

import (
	"encoding/gob"
	"log"
	"os"
)

const (
	WEBCA_CFG  = ".webca.cfg"
	WEBCA_FILE = "webca.pem"
	WEBCA_KEY  = "webca.key.pem"
)

type User struct {
	Username, Fullname, Password, Email string
}

type config struct {
	Mailer
	advance   int // days before the cert. expires that the notification will be sent
	users     map[string]*User
	certs     CertTree
	user2cert map[string]string
	cert2user map[string]string
}

type Configurer interface {
	tlsFiles() (cert, key string, ok bool)
	save()
}

func LoadConfig() Configurer {
	_, err := os.Stat(WEBCA_CFG)
	if os.IsNotExist(err) {
		return nil
	}
	f, err := os.Open(WEBCA_CFG)
	handleFatal(err)
	defer f.Close()
	dec := gob.NewDecoder(f)
	if dec == nil {
		log.Fatalf("(Warning) Could not decode " + WEBCA_CFG + "!")
	}
	cfg := config{}
	err = dec.Decode(&cfg)
	handleFatal(err)
	return &cfg
}

func (cfg *config) save() {
	f, err := os.OpenFile(WEBCA_CFG, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	handleFatal(err)
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(cfg)
	handleFatal(err)
}

func (cfg *config) tlsFiles() (cert, key string, ok bool) {
	ok = true
	cert = WEBCA_FILE
	_, err := os.Stat(cert)
	if os.IsNotExist(err) {
		cert = ""
		ok = false
	}
	key = WEBCA_FILE
	_, err = os.Stat(key)
	if os.IsNotExist(err) {
		key = ""
		ok = false
	}
	return cert, key, ok
}

