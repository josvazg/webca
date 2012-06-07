package main

import (
	"code.google.com/p/gorilla/context"
	"code.google.com/p/gorilla/sessions"
	//"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var store = sessions.NewCookieStore([]byte("34534askjdfhkjsd41234rrf34856"))

var templates = template.Must(template.ParseFiles("html/setup.html", "html/newuser.html",
	"html/newca.html", "html/newcert.html",
	"html/templates.html", "html/style.css", "html/translate_en.html"))

var defaultHandler func (w http.ResponseWriter, r *http.Request)

const (
	SETUPADDR    = "127.0.0.1:80"
	ALTSETUPADDR = "127.0.0.1:9090"
)

type SetupWizard struct {
	Step   int
	U      *User
	M      *Mailer
	Server string
	Port   string
}

type PageStatus struct {
	SetupWizard
	Error string
}

func (ps *PageStatus) Tr(s string) string {
	return tr(s)
}

// Transaltion function
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
	ps := &PageStatus{SetupWizard: SetupWizard{Step: 1}}
	context.DefaultContext.Set(r, "pageStatus", ps)
	r.URL,_=r.URL.Parse("/newuser")
	autoPage(w, r)
}

func userSetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{SetupWizard: SetupWizard{Step: 1, U:&User{}}}
	ps.Step, _ = strconv.Atoi(r.FormValue("Step"))
	ps.U.Username = r.FormValue("Username")
	ps.U.Fullname = r.FormValue("Fullname")
	ps.U.Email = r.FormValue("Email")
	pwd := r.FormValue("Password")
	pwd2 := r.FormValue("Password2")
	if pwd == "" || pwd != pwd2 {
		ps.Error = tr("BadPasswd")
		context.DefaultContext.Set(r, "pageStatus", ps)
		r.URL,_=r.URL.Parse("/newuser")
		autoPage(w, r)
		return
	}
	ps.U.Password = pwd
	ps.Step += 1
	log.Println(ps.U)
	session, _ := store.Get(r, "")
	_,ok:=session.Values["ps"]
	if ok {
		session.Values["ps"].(*PageStatus).U=ps.U
	} else {
		session.Values["ps"] = ps
	}
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

func saveSetup(w http.ResponseWriter, r *http.Request) {
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
	r.URL,_=r.URL.Parse("/newca")
	autoPage(w, r)
}

func autoPage(w http.ResponseWriter, r *http.Request) {
	ps:=getPageStatus(r)
	page:=page(r)
	if page=="" {
		defaultHandler(w,r)
		return
	}
	if !checkPage(page) {
		http.NotFound(w, r)
		return
	}
	log.Println("uri=",r.URL.RequestURI(),"page=",page)
	err := templates.ExecuteTemplate(w, page+".html", ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func checkPage(page string) bool {
	if page=="templates" {
		return false
	}
	_, err := os.Stat("html/"+page+".html")
	if os.IsNotExist(err) {
		return false
	}
	return true

}

func page(r *http.Request) string {
	pg:=r.URL.RequestURI()
	if strings.Contains(pg, "?") {
		pg=strings.Split(pg, "?")[0]
	}
	if strings.HasPrefix(pg, "/") {
		pg=pg[1:]
	}
	if strings.Contains(pg, "/") {
		parts:=strings.Split(pg, "/")
		pg=parts[len(parts)-1]
	}
	if strings.Contains(pg, ".") {
		pg=strings.Split(pg,".")[0]
	}
	return pg
}

func getPageStatus(r *http.Request) *PageStatus {
	ips:=context.DefaultContext.Get(r, "pageStatus")
	if ips==nil {
	    session, _ := store.Get(r, "webca")
	    ips=session.Values["pageStatus"]
	}
	if ips!=nil {
		return ips.(*PageStatus)
	}
	return &PageStatus{}
}

func setup() {
	log.Printf("(Warning) Starting setup, go to http://127.0.0.1/...")
	smux := http.NewServeMux()
	setupServer := http.Server{Addr: SETUPADDR, Handler: smux}
	defaultHandler=startSetup
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

