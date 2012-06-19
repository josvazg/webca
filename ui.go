package webca

import (
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	ADDR      = ""
	SETUPADDR = "127.0.0.1"
	PORT      = 443
	SETUPPORT = 80
	ALTPORT   = 8000
	_HTML     = ".html"
)

// templates contains all web templates
var templates *template.Template

// templateIndex contains a quick way to test page template existence
var templateIndex map[string]*template.Template

// defaultHandler points to the handler for '/' requests
var defaultHandler func(w http.ResponseWriter, r *http.Request)

// CertSetup contains the config to generate a certificate
type CertSetup struct {
	Name     pkix.Name
	Duration int
}

// PageStatus contains all values that a page and its templates need 
// (including the SetupWizard when the setup is running)
//	U      User
// SetupWizard contains the status of the setup wizard and may be included in the PageStatus Map
//	CA     CertSetup
//	Cert   CertSetup
//	M      Mailer
type PageStatus map[string]interface{}

// oneSetup holds the setup lock
var oneSetup sync.Mutex

// setupDone tells whether the configation has been applied or not
var setupDone bool

// init prepares all web templates before anything else
func init() {
	templates = template.New("webcaTemplates")
	templates.Funcs(template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"tr": tr, "indexOf": indexOf,
	})
	template.Must(templates.ParseFiles("html/mailer.html", "html/user.html",
		"html/ca.html", "html/cert.html", "html/setup.html", "html/setupDone.html",
		"html/templates.html", "html/templates.js", "html/style.css"))

	// build templateIndex
	templateIndex = make(map[string]*template.Template)

	for _, t := range templates.Templates() {
		//log.Println("template: ", t.Name())
		if strings.HasSuffix(t.Name(), _HTML) {
			templateIndex[t.Name()] = t
		}
	}
}

// LoadCrt loads variables "Prfx" and "Crt" into PageSetup to point to the right 
// CertSetup and its prefix and sets a default duration for that cert
func (ps PageStatus) LoadCrt(arg interface{}, prfx string, defaultDuration int) string {
	cs := arg.(*CertSetup)
	ps["Crt"] = cs
	ps["Prfx"] = prfx
	cs.Duration = defaultDuration
	return ""
}

// IsDuration returns whether or not the given duration is the selected one on the loaded Crt
func (ps PageStatus) IsSelected(duration int) bool {
	crt := ps["Crt"]
	if crt == nil {
		return false
	}
	cs := crt.(*CertSetup)
	return cs.Duration == duration
}

// tr is the app translation function
func tr(s string) string {
	return s
}

// indexOf allows to access strings on a string array
func indexOf(sa []string, index int) string {
	if sa == nil || len(sa) < (index+1) {
		return ""
	}
	return sa[index]
}

// WebCA starts the prepares and serves the WebApp 
func WebCA() {
	addr := PrepareServer()
	log.Printf("Go to http://" + addr + "/...")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Printf("Could not start server on address '"+addr+"'!: %s", err)
	}
	addr = alternateAddress(addr)
	log.Printf("(Warning) Failed to listen, go to http://" + addr + "/...")
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatalf("Could not start!: %s", err)
	}
}

// alternateAddress returns the alternate address by changing or adding the port to ALTPORT
func alternateAddress(addr string) string {
	if strings.Contains(addr, ":") {
		parts := strings.Split(addr, ":")
		addr = parts[0]
	}
	return fmt.Sprintf("%s:%v", addr, ALTPORT)
}

// prepareServer prepares the Web handlers for the setup wizard if there is no HTTPS config or 
// the normal app if the app is already configured
func PrepareServer() string {
	// load config...
	cfg := LoadConfig()
	if cfg == nil { // if config is null then run the setup
		addr := fmt.Sprintf("%s:%v", SETUPADDR, SETUPPORT)
		log.Printf("(Warning) Starting WebCA setup...")
		http.HandleFunc("/", showSetup)
		http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
		http.Handle("/crt/", http.StripPrefix("/crt/", certServer(http.Dir("."))))
		http.HandleFunc("/setup", setup)
		http.HandleFunc("/restart", restart)
		return addr
	}
	// otherwise start the normal app
	log.Printf("WebCA normal startup...\n")
	return ADDR
}

// certServer returns a certificate server filtering the downloadable cert files properly
func certServer(dir http.Dir) http.Handler {
	h := http.FileServer(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".key.pem") || !strings.HasSuffix(r.URL.Path, ".pem") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-disposition", "attachment; filename="+r.URL.Path)
		w.Header().Set("Content-type", "application/x-pem-file")
		h.ServeHTTP(w, r)
	})
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
	log.Printf("cmd: %v\n", flag.Args())
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

// readUser reads the user data from the request
func readUser(r *http.Request) User {
	u := User{}
	u.Username = r.FormValue("Username")
	u.Fullname = r.FormValue("Fullname")
	u.Email = r.FormValue("Email")
	u.Password = r.FormValue("Password")
	return u
}

// readCertSetup reads the certificate setup from the request
func readCertSetup(prefix string, r *http.Request) (*CertSetup, error) {
	cs := CertSetup{}
	prepareName(&cs.Name)
	cs.Name.CommonName = r.FormValue(prefix + ".CommonName")
	cs.Name.StreetAddress[0] = r.FormValue(prefix + ".StreetAddress")
	cs.Name.PostalCode[0] = r.FormValue(prefix + ".PostalCode")
	cs.Name.Locality[0] = r.FormValue(prefix + ".Locality")
	cs.Name.Province[0] = r.FormValue(prefix + ".Province")
	cs.Name.OrganizationalUnit[0] = r.FormValue(prefix + ".OrganizationalUnit")
	cs.Name.Organization[0] = r.FormValue(prefix + ".Organization")
	cs.Name.Country[0] = r.FormValue(prefix + ".Country")
	duration, err := strconv.Atoi(r.FormValue(prefix + ".Duration"))
	if err != nil || duration < 0 {
		return nil, fmt.Errorf("%s: %v", tr("Wrong duration!"), err)
	}
	cs.Duration = duration
	return &cs, nil
}

// readMailer reads the mailer config from the request
func readMailer(r *http.Request) Mailer {
	m := Mailer{}
	m.User = r.FormValue("M.User")
	m.Server = r.FormValue("M.Server")
	port := r.FormValue("M.Port")
	if port != "" {
		m.Server += ":" + port
	}
	m.Passwd = r.FormValue("M.Password")
	return m
}

// autoPage displays the page specified in the URL that matches a template
func autoPage(w http.ResponseWriter, r *http.Request) {
	page := page(r)
	if page == "" {
		defaultHandler(w, r)
		return
	}
	if !checkPage(page) {
		http.NotFound(w, r)
		return
	}
	//log.Println("uri=", r.URL.RequestURI(), "page=", page, "ps=", ps)
	err := templates.ExecuteTemplate(w, page+_HTML, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// checkPage tells whether the given page does have a template
func checkPage(page string) bool {
	if page == "templates" {
		return false
	}
	_, ok := templateIndex[page+_HTML]
	return ok
}

// page extracts the page the user wants to go to from the URL
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
