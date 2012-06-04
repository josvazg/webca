package main

import (
	//"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

var templates = template.Must(template.ParseFiles("html/setup.html", "html/newca.html",
	"html/newcert.html",
	"html/templates.html", "html/style.css", "html/translate_en.html"))

const (
	WEBCA_FILE   = "webca.pem"
	WEBCA_KEY    = "webca.key.pem"
	ALTSETUPADDR = ":9090"
)

// webca starts the setup if there is no HTTPS config or the normal app if it is present
func webca() {
	if !hasTLSConfig() {
		setupHttps()
	}
}

// hasTLSConfig checks whethere there is configuration files for HTTPS or not
func hasTLSConfig() bool {
	_, err := os.Stat(WEBCA_FILE)
	if os.IsNotExist(err) {
		return false
	}
	_, err = os.Stat(WEBCA_KEY)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func sayhi(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "Hi there!")
	err := templates.ExecuteTemplate(w, "setup.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newca(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "newca.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newcert(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "newcert.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func setupHttps() {
	log.Printf("(Warning) Starting setup, go to http://localhost/...")
	smux := http.NewServeMux()
	setupServer := http.Server{Handler: smux}
	smux.HandleFunc("/", sayhi)
	smux.HandleFunc("/newca", newca)
	smux.HandleFunc("/newcert", newcert)
	err := setupServer.ListenAndServe()
	if err != nil && !strings.Contains(err.Error(), "perm") {
		log.Fatalf("Could not start setup!: %s", err)
	}
	setupServer.Addr = ALTSETUPADDR
	log.Printf("(Warning) Failed to listen on port :80 go to http://localhost" +
		ALTSETUPADDR + "/...")
	err = setupServer.ListenAndServe()
	if err != nil {
		log.Fatalf("Could not start setup!: %s", err)
	}
}

func main() {
	webca()
}

