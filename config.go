package webca

import (
	"encoding/gob"
	"log"
	"os"
	"sync"
)

const (
	WEBCA_CFG = ".webca.cfg"
)

// oneCfg ensures serialized access to configuration
var oneCfg sync.RWMutex

// cachedCfg prevents from reading the config from file too many times
var cachedCfg *config

// User contains the App's User details
type User struct {
	Username, Fullname, Password, Email string
}

// config contains the App's Configuration
type config struct {
	Mailer  *Mailer
	Advance int // days before the cert. expires that the notification will be sent
	Users   map[string]User
	WebCert *Cert
}

// New Config creates a new Config
func NewConfig(u User, cacert *Cert, cert *Cert, m Mailer) *config {
	log.Println("cert=", cert)
	cfg := &config{Mailer: &m, Advance: 15, Users: make(map[string]User), WebCert: cert}
	cfg.Users[u.Username] = u
	log.Println("New Cfg=", cfg)
	return cfg
}

// LoadConfig (re)loads a config
// (It needs to be thread safe)
func LoadConfig() *config {
	oneCfg.RLock()
	defer oneCfg.RUnlock()
	if cachedCfg != nil {
		return cachedCfg
	}
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
	cfg := &config{}
	err = dec.Decode(cfg)
	handleFatal(err)
	cachedCfg = cfg
	return cfg
}

// Save puts the config state into persistent storage
// (It needs to be thread safe)
func (cfg *config) Save() error {
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

// WebCert returns the current Web Certificate
func (cfg *config) getWebCert() Cert {
	return *cfg.WebCert
}

// User returns a copy of the User named username
func (cfg *config) getUser(username string) User {
	return cfg.Users[username]
}

// crypt transforms a password to a hashed form avoiding storing it in clear text
func crypt(passwd string) string {
	return passwd // TODO decide password encryption later (bcrypt?)
}

