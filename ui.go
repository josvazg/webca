package webca

import (
	"crypto/x509/pkix"
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
	SETUPADDR    = "127.0.0.1:80"
	ALTSETUPADDR = "127.0.0.1:9090"
	_HTML        = ".html"
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

// SetupWizard contains the status of the setup wizard and may be included in the PageStatus Map
//	CA     CertSetup
//	Cert   CertSetup
//	M      Mailer

// PageStatus contains all values that a page and its templates need 
// (including the SetupWizard when the setup is running)
//	U     User
type PageStatus map[string]interface{}

// configuration holds the config lock
var configuration sync.Mutex

// configurationDone tells whether the configation has been applied or not
var configurationDone bool

// init prepares all web templates before anything else
func init() {
	templates = template.New("webcaTemplates")
	templates.Funcs(template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"tr": tr, "indexOf": indexOf,
	})
	template.Must(templates.ParseFiles("html/mailer.html", "html/user.html",
		"html/ca.html", "html/cert.html", "html/setup.html",
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

// WebCA starts the setup if there is no HTTPS config or the normal app if it is present
func WebCA() {
	// load config to run the normal app or the setup wizard
	cfg := LoadConfig()
	if cfg == nil {
		setup()
	}
}

// startSetup starts the setup wizard form
func startSetup(w http.ResponseWriter, r *http.Request) {
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

// endSetup checks and saves the initial setup from the wizard form
func endSetup(w http.ResponseWriter, r *http.Request) {
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
	configuration.Lock()
	defer configuration.Unlock()
	if !configurationDone {
		configure(user, certs["CA"], certs["Cert"], mailer, w, r)
		configurationDone = true
		fmt.Fprintln(w, "Setup OK!")
	} else {
		fmt.Fprintln(w, "Setup already done!")
	}
}

// configure gets the config data and prepares certificates and the config file
func configure(user User, ca, c *CertSetup, mailer Mailer,
	w http.ResponseWriter, r *http.Request) {
	log.Println("Running setup...")
	log.Println("user=", user)
	log.Println("ca=", ca)
	log.Println("c=", c)
	log.Println("mailer=", mailer)
	cacert, err := GenCACert(ca.Name, ca.Duration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	cert, err := GenCert(cacert, c.Name.CommonName, c.Duration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	log.Println("CA", cacert)
	log.Println("Cert", cert)
	copyTo(cacert.crt.Subject.CommonName+".pem", "cert.pem")
	copyTo(cert.crt.Subject.CommonName+".pem", "cert.pem")
	copyTo(cert.crt.Subject.CommonName+".key.pem", "key.pem")
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

// setup starts the webca "setup wizard"
func setup() {
	log.Printf("(Warning) Starting setup, go to http://127.0.0.1/...")
	smux := http.NewServeMux()
	setupServer := http.Server{Addr: SETUPADDR, Handler: smux}
	RegisterSetup(smux)
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

// RegisterSetup register just setup handlers
func RegisterSetup(smux *http.ServeMux) {
	defaultHandler = startSetup
	smux.HandleFunc("/", autoPage)
	smux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	smux.HandleFunc("/endSetup", endSetup)
}

