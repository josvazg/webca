package webca

import (
	"encoding/gob"
	"log"
	"os"
	"sync"
)

const (
	WEBCA_CFG  = ".webca.cfg"
	WEBCA_FILE = "webca.pem"
	WEBCA_KEYFILE  = "webca.key.pem"
)

// oneCfg ensures serialized access to configuration
var oneCfg sync.Mutex

// User contains the App's User details
type User struct {
	Username, Fullname, Password, Email string
}

// config contains the App's Configuration
type config struct {
	Mailer  *Mailer
	Advance int // days before the cert. expires that the notification will be sent
	Users   map[string]*User
	Certs   *CertTree
	WebCert *Cert
}

// Configurer is the interface to access and user Configuration
type Configurer interface {
	save() error
	webCert() *Cert
}

// New Config obtains a new Config
func NewConfig(u User, cacert *Cert, cert *Cert, m Mailer) Configurer {
	certs := newCertTree()
	certs.addCert(cacert)
	certs.addCert(cert)
	cfg := &config{&m, 15, make(map[string]*User), certs, cert}
	cfg.Users[u.Username] = &u
	return cfg
}

// LoadConfig (re)loads a config
// (It needs to be thread safe)
func LoadConfig() Configurer {
	oneCfg.Lock()
	defer oneCfg.Unlock()
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

// save puts the current config into persistent storage
// (It needs to be thread safe)
func (cfg *config) save() error {
	oneCfg.Lock()
	defer oneCfg.Unlock()
	f, err := os.OpenFile(WEBCA_CFG, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Println("can't open")
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(cfg)
	if err != nil {
		log.Println("can't save")
		return err
	}
	return nil
}

// webCA returns this web CA Cert
func (cfg *config) webCert() *Cert {
	return cfg.WebCert
}
