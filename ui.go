package main

import (
	"code.google.com/p/gorilla/context"
	"code.google.com/p/gorilla/sessions"
	//"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var store = sessions.NewCookieStore([]byte("webcasecret"))

var templates = template.Must(template.ParseFiles("html/setup.html", "html/newuser.html",
	"html/newca.html", "html/newcert.html",
	"html/templates.html", "html/style.css", "html/translate_en.html"))

const (
	SETUPADDR    = "127.0.0.1:80"
	ALTSETUPADDR = "127.0.0.1:9090"
)

type SetupWizard struct {
	Step   int
	U      User
	M      Mailer
	Server string
	Port   string
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

func usersetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{}
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
	ps.Step += 1
	log.Println(ps.U)
	context.DefaultContext.Set(r, "pageStatus", ps)
	session, _ := store.Get(r, "")
	session.Values["user"] = ps.U
	newsetup(w, r)
}

func newsetup(w http.ResponseWriter, r *http.Request) {
	ps := context.DefaultContext.Get(r, "pageStatus").(*PageStatus)
	if ps.M.User == "" {
		ps.M.User = ps.U.Email
	}
	err := templates.ExecuteTemplate(w, "setup.html", ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func savesetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{}
	ps.Step, _ = strconv.Atoi(r.FormValue("Step"))
	ps.M.User = r.FormValue("Email")
	ps.Server = r.FormValue("Server")
	ps.Port = r.FormValue("Port")
	pwd := r.FormValue("Password")
	pwd2 := r.FormValue("Password2")
	if pwd == "" || pwd != pwd2 {
		ps.Error = tr("BadPasswd")
		context.DefaultContext.Set(r, "pageStatus", ps)
		newsetup(w, r)
		return
	}
	ps.M.Server = ps.Server
	if ps.Port != "" {
		ps.M.Server += ":" + ps.Port
	}
	ps.M.Passwd = pwd
	log.Println(ps.M)
	context.DefaultContext.Set(r, "pageStatus", ps)
	session, _ := store.Get(r, "")
	session.Values["mailer"] = ps.M
	newca(w, r)
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
	log.Printf("(Warning) Starting setup, go to http://127.0.0.1/...")
	smux := http.NewServeMux()
	setupServer := http.Server{Addr: SETUPADDR, Handler: smux}
	smux.HandleFunc("/", startSetup)
	smux.HandleFunc("/usersetup", usersetup)
	smux.HandleFunc("/newsetup", newsetup)
	smux.HandleFunc("/saveSetup", savesetup)
	smux.HandleFunc("/newca", newca)
	smux.HandleFunc("/newcert", newcert)
	err := setupServer.ListenAndServe()
	if err != nil && !strings.Contains(err.Error(), "perm") {
		log.Fatalf("Could not start setup!: %s", err)
	}
	setupServer.Addr = ALTSETUPADDR
	log.Printf("(Warning) Failed to listen on port :80 go to http://" + ALTSETUPADDR + "/...")
	err = setupServer.ListenAndServe()
	if err != nil {
		log.Fatalf("Could not start setup!: %s", err)
	}
}

func main() {
	webca()
}

