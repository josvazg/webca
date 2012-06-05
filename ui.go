package main

import (
	"code.google.com/p/gorilla/context"
	//"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var templates = template.Must(template.ParseFiles("html/setup.html", "html/newuser.html",
	"html/newca.html", "html/newcert.html",
	"html/templates.html", "html/style.css", "html/translate_en.html"))

const (
	ALTSETUPADDR = ":9090"
)

type SetupWizard struct {
	Step int
	U    User
	M    Mailer
}

type PageStatus struct {
	SetupWizard
	Error string
}

func tr(s string) string {
	return s
}

// webca starts the setup if there is no HTTPS config or the normal app if it is present
func webca() {
	cfg := LoadConfig()
	if cfg == nil {
		setup()
	}
}

func startSetup(w http.ResponseWriter, r *http.Request) {
	ps := PageStatus{SetupWizard: SetupWizard{Step: 1}}
	context.DefaultContext.Set(r, "pageStatus", ps)
	newuser(w, r)
}

func newuser(w http.ResponseWriter, r *http.Request) {
	ps := context.DefaultContext.Get(r, "pageStatus")
	err := templates.ExecuteTemplate(w, "newuser.html", ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func userSetup(w http.ResponseWriter, r *http.Request) {
	ps := PageStatus{}
	ps.Step, _ = strconv.Atoi(r.FormValue("Step"))
	ps.U.Username = r.FormValue("Username")
	ps.U.Fullname = r.FormValue("Fullname")
	ps.U.Email = r.FormValue("Email")
	pwd := r.FormValue("Password")
	pwd2 := r.FormValue("Password2")
	if pwd == "" || pwd != pwd2 {
		ps.Error = tr("BadPasswd")
		context.DefaultContext.Set(r, "pageStatus", ps)
		newuser(w, r)
		return
	}
	ps.U.Password = pwd

	log.Println(ps.U)
	context.DefaultContext.Set(r, "pageStatus", ps)
	newsetup(w, r)
}

func newsetup(w http.ResponseWriter, r *http.Request) {
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

func setup() {
	log.Printf("(Warning) Starting setup, go to http://localhost/...")
	smux := http.NewServeMux()
	setupServer := http.Server{Handler: smux}
	smux.HandleFunc("/", startSetup)
	smux.HandleFunc("/userSetup", userSetup)
	smux.HandleFunc("/newsetup", newsetup)
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

