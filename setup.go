package webca

import (
	"crypto/x509/pkix"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
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

// PrepareSetup prepares the Web handlers for the setup wizard
func PrepareSetup() string {
	addr := fmt.Sprintf("%s:%v", SETUPADDR, SETUPPORT)
	log.Printf("(Warning) Starting WebCA setup...")
	http.HandleFunc("/", showSetup)
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	http.Handle("/crt/", http.StripPrefix("/crt/", certServer(http.Dir("."))))
	http.HandleFunc("/setup", setup)
	http.HandleFunc("/restart", restart)
	return addr
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
	log.Printf("Checking whether to do setup or not...")
	ps := PageStatus{}
	ps["CAName"] = certs["CA"].Name.CommonName
	oneSetup.Lock()
	defer oneSetup.Unlock()
	if !setupDone {
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
		ps["Message"] = tr("Setup OK!")
	} else {
		ps["Message"] = tr("Setup already done!")
	}
	err := templates.ExecuteTemplate(w, "setupDone"+_HTML, ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// restart stops the app and restarts into webca if possible
func restart(w http.ResponseWriter, r *http.Request) {
	if !setupDone {
		http.Error(w, tr("Can't restart, setup wasn't done!"), http.StatusInternalServerError)
	}
	if LoadConfig() == nil {
		http.Error(w, tr("Can't restart, there is no config to load!"),
			http.StatusInternalServerError)
	}
	var cmd *exec.Cmd
	if len(os.Args) > 1 {
		cmd = exec.Command(os.Args[0], os.Args[1:]...)
	} else {
		cmd = exec.Command(os.Args[0])
	}
	oneSetup.Lock()
	defer oneSetup.Unlock()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	log.Print("Restarting: ", cmd.Args)
	if err := cmd.Start(); err != nil {
		log.Print("error:", err)
		return
	} else {
		log.Println("Setup process ends")
		err = templates.ExecuteTemplate(w, "restart"+_HTML, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		os.Exit(0)
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
