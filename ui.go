package main

import (
	"code.google.com/p/gorilla/context"
	"code.google.com/p/gorilla/sessions"
	//"fmt"
	"html/template"
	"log"
	"net/http"
	//"os"
	"strconv"
	"strings"
)

const (
	SETUPADDR    = "127.0.0.1:80"
	ALTSETUPADDR = "127.0.0.1:9090"
	PAGESTATUS   = "pageStatus"
	_HTML        = ".html"
)

// store for web sessions
var store = sessions.NewCookieStore([]byte("34534askjdfhkjsd41234rrf34856"))

// templates contains all web templates
var templates = template.Must(template.ParseFiles("html/setup.html", "html/newuser.html",
	"html/newca.html", "html/newcert.html",
	"html/templates.html", "html/style.css", "html/translate_en.html"))

// templateIndex contains a quick way to test page template existence
var templateIndex map[string]*template.Template

// defaultHandler points to the handler for '/' requests
var defaultHandler func(w http.ResponseWriter, r *http.Request)

// SetupWizard contains the status of the setup wizard
type SetupWizard struct {
	Step   int
	U      *User
	M      *Mailer
	Server string
	Port   string
}

// PageStatus contains all values that a page and its templates need (including the SetupWizard)
type PageStatus struct {
	SetupWizard
	Error string
}

// Tr allows to access tr from templates as well
func (ps *PageStatus) Tr(s string) string {
	return tr(s)
}

// tr is the app translation function
func tr(s string) string {
	return s
}

// webca starts the setup if there is no HTTPS config or the normal app if it is present
func webca() {
	// build templateIndex
	templateIndex = make(map[string]*template.Template)
	for _, t := range templates.Templates() {
		if strings.HasSuffix(t.Name(), _HTML) {
			templateIndex[t.Name()] = t
		}
	}
	// load config to run the normal app or the setup wizard
	cfg := LoadConfig()
	if cfg == nil {
		setup()
	}
}

// startSetup starts the setup wizard web page sequence
func startSetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{SetupWizard: SetupWizard{Step: 1, U: &User{}}}
	forwardTo(w, r, ps, "newuser")
}

// userSetup saves a new user
func userSetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{SetupWizard: SetupWizard{Step: 1, U: &User{}}}
	ps.Step, _ = strconv.Atoi(r.FormValue("Step"))
	ps.U.Username = r.FormValue("Username")
	ps.U.Fullname = r.FormValue("Fullname")
	ps.U.Email = r.FormValue("Email")
	pwd := r.FormValue("Password")
	pwd2 := r.FormValue("Password2")
	if pwd == "" {
		ps.Error = tr("Password is empty!")
		forwardTo(w, r, ps, "newuser")
		return
	} else if pwd != pwd2 {
		ps.Error = tr("Passwords don't match!")
		forwardTo(w, r, ps, "newuser")
		return
	}
	ps.U.Password = pwd
	ps.Step += 1
	log.Println(ps.U)
	session, _ := store.Get(r, "")
	_, ok := session.Values[PAGESTATUS]
	if ok {
		session.Values[PAGESTATUS].(*PageStatus).U = ps.U
	} else {
		session.Values[PAGESTATUS] = ps
	}
	forwardTo(w, r, nil, "newca")
}

// mailerSetup configures the mailer settings
func mailerSetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{}
	ps.Step, _ = strconv.Atoi(r.FormValue("Step"))
	ps.M.User = r.FormValue("Email")
	ps.Server = r.FormValue("Server")
	ps.Port = r.FormValue("Port")
	pwd := r.FormValue("Password")
	pwd2 := r.FormValue("Password2")
	if pwd == "" || pwd != pwd2 {
		ps.Error = tr("BadPasswd")
		forwardTo(w, r, ps, "newsetup")
		return
	}
	ps.M.Server = ps.Server
	if ps.Port != "" {
		ps.M.Server += ":" + ps.Port
	}
	ps.M.Passwd = pwd
	log.Println(ps.M)
	context.DefaultContext.Set(r, PAGESTATUS, ps)
	session, _ := store.Get(r, "")
	_, ok := session.Values[PAGESTATUS]
	if ok {
		session.Values[PAGESTATUS].(*PageStatus).M = ps.M
	} else {
		session.Values[PAGESTATUS] = ps
	}
	// TODO finish setup and start the webca
}

// forwardTo passes control to the given page
func forwardTo(w http.ResponseWriter, r *http.Request, ps *PageStatus, page string) {
	if ps != nil {
		context.DefaultContext.Set(r, PAGESTATUS, ps)
	}
	r.URL, _ = r.URL.Parse("/" + page)
	autoPage(w, r)
}

// autoPage loads current pageStatus and then displays the page given in the URL
func autoPage(w http.ResponseWriter, r *http.Request) {
	ps := getPageStatus(r)
	page := page(r)
	if page == "" {
		defaultHandler(w, r)
		return
	}
	if !checkPage(page) {
		http.NotFound(w, r)
		return
	}
	log.Println("uri=", r.URL.RequestURI(), "page=", page)
	err := templates.ExecuteTemplate(w, page+_HTML, ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// checkPage tells whether the given page does really exist or not
func checkPage(page string) bool {
	if page == "templates" {
		return false
	}
	_, ok := templateIndex[page+_HTML]
	return ok
}

// page extracts the page the user whats to go to from the URL
func page(r *http.Request) string {
	pg := r.URL.RequestURI()
	if strings.Contains(pg, "?") {
		pg = strings.Split(pg, "?")[0]
	}
	if strings.HasPrefix(pg, "/") {
		pg = pg[1:]
	}
	if strings.Contains(pg, "/") {
		parts := strings.Split(pg, "/")
		pg = parts[len(parts)-1]
	}
	if strings.Contains(pg, ".") {
		pg = strings.Split(pg, ".")[0]
	}
	return pg
}

// getPageStatus loads current pageStatus (from request or session)
func getPageStatus(r *http.Request) *PageStatus {
	ips := context.DefaultContext.Get(r, PAGESTATUS)
	if ips == nil {
		session, _ := store.Get(r, "webca")
		ips = session.Values[PAGESTATUS]
	}
	if ips != nil {
		return ips.(*PageStatus)
	}
	return &PageStatus{}
}

// setup starts the webca "setup wizard"
func setup() {
	log.Printf("(Warning) Starting setup, go to http://127.0.0.1/...")
	smux := http.NewServeMux()
	setupServer := http.Server{Addr: SETUPADDR, Handler: smux}
	defaultHandler = startSetup
	smux.HandleFunc("/", autoPage)
	smux.HandleFunc("/userSetup", userSetup)
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

