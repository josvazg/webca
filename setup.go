package webca

import (
	"crypto/x509/pkix"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

const (
	SETUPADDR = "127.0.0.1"
	SETUPPORT = 80
)

// CertSetup contains the config to generate a certificate
type CertSetup struct {
	Name     pkix.Name
	Duration int
}

// oneSetup holds the setup lock
var oneSetup sync.Mutex

// setupDone tells whether the configation has been applied or not
var setupDone bool

// rootFunc points to the root function that is different on setup and normal mode
var rootFunc func(w http.ResponseWriter, r *http.Request)

// PrepareSetup prepares the Web handlers for the setup wizard
func PrepareSetup() string {
	addr := fmt.Sprintf("%s:%v", SETUPADDR, SETUPPORT)
	log.Printf("(Warning) Starting WebCA setup...")
	rootFunc=showSetup
	http.HandleFunc("/", smartSwitch)
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	http.Handle("/crt/", http.StripPrefix("/crt/", certServer(http.Dir("."))))
	http.HandleFunc("/setup", setup)
	http.HandleFunc("/restart", restart)
	return addr
}

// smartSwitch redirects to showSetup or index depending on whether the setup is done or not
func smartSwitch(w http.ResponseWriter, r *http.Request) {
	rootFunc(w,r)
}

// showSetup shows the setup wizard form
func showSetup(w http.ResponseWriter, r *http.Request) {
	ps := PageStatus{
		"Server": "smtp.gmail.com",
		"Port":   "587",
		"CA":     &CertSetup{},
		"Cert":   &CertSetup{},
		"U":      &User{},
		"M":      &Mailer{},
	}
	err := templates.ExecuteTemplate(w, "setup"+_HTML, ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// setup checks and saves the initial setup from the wizard form
func setup(w http.ResponseWriter, r *http.Request) {
	log.Printf("Checking whether to do setup or not...")
	oneSetup.Lock()
	defer oneSetup.Unlock()
	if !setupDone {
		user := readUser(r)
		certs := make(map[string]*CertSetup, 2)
		for _, prefix := range []string{"CA", "Cert"} {
			crt, err := readCertSetup(prefix, r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			certs[prefix] = crt
		}
		mailer := readMailer(r)
		ca, c := certs["CA"], certs["Cert"]
		log.Printf("Running setup...\nuser=%s\nca=%s\nc=%s\nmailer%s\n", user, ca, c, mailer)
		cacert, err := GenCACert(ca.Name, ca.Duration)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cert, err := GenCert(cacert, c.Name.CommonName, c.Duration)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("CA=%s\nCert=%s\n", cacert, cert)
		copyTo(cacert.Crt.Subject.CommonName+".pem", WEBCA_FILE)
		copyTo(cert.Crt.Subject.CommonName+".pem", WEBCA_FILE)
		copyTo(cert.Crt.Subject.CommonName+".key.pem", WEBCA_KEY)
		log.Printf("Saving config...")
		if err = NewConfig(user, cacert, cert, mailer).save(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setupDone = true
		rootFunc=restart
	}
	restart(w,r)
}

// restart tells the user the setup is already done so she can proceed to the WebCA
func restart(w http.ResponseWriter, r *http.Request) {
	cfg:=LoadConfig()
	ps := PageStatus{}
	ps["Message"] = tr("Setup is done!")
	ps["CAName"] = cfg.webCA().Crt.Subject.CommonName
	err := templates.ExecuteTemplate(w, "restart"+_HTML, ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// copyTo copies from file orig to file dest, appending if dest exists
func copyTo(orig, dest string) error {
	r, err := os.Open(orig)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := os.OpenFile(dest, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0640)
	if err != nil {
		return err
	}
	defer w.Close()
	if _, err = io.Copy(w, r); err != nil {
		return err
	}
	return nil
}
