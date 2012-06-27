package webca

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	PORT    = 443
	PORTFIX = 8000
	_HTML   = ".html"
)

// address is a complex bind address
type address struct {
	addr, certfile, keyfile string
	tls                     bool
}

// portFix contains the port correction when low ports are not permited
var portFix int

// listenAndServe starts the server with or without TLS on the address
func (a address) listenAndServe(smux *http.ServeMux) error {
	if a.tls {
		return http.ListenAndServeTLS(a.addr, a.certfile, a.keyfile, smux)
	}
	return http.ListenAndServe(a.addr, smux)
}

// String prints this address properly
func (a address) String() string {
	prefix := "http"
	if a.tls {
		prefix = "https"
	}
	return prefix + "://" + a.addr
}

// templates contains all web templates
var templates *template.Template

// templateIndex contains a quick way to test page template existence
var templateIndex map[string]*template.Template

// defaultHandler points to the handler for '/' requests
var defaultHandler func(w http.ResponseWriter, r *http.Request)

// PageStatus contains all values that a page and its templates need 
// (including the SetupWizard when the setup is running)
//	U      User
// SetupWizard contains the status of the setup wizard and may be included in the PageStatus Map
//	CA     CertSetup
//	Cert   CertSetup
//	M      Mailer
type PageStatus map[string]interface{}

// init prepares all web templates before anything else
func init() {
	templates = template.New("webcaTemplates")
	templates.Funcs(template.FuncMap{
		// The name "title" is what the function will be called in the template text.
		"tr": tr, "indexOf": indexOf,
	})
	template.Must(templates.ParseFiles("html/index.html",
		"html/mailer.html", "html/user.html",
		"html/ca.html", "html/cert.html",
		"html/setup.html", "html/restart.html",
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
	smux := http.DefaultServeMux
	addr := PrepareServer(smux)
	err := addr.listenAndServe(smux)
	if portFix == 0 { // port Fixing is only applied once
		if err != nil {
			log.Printf("Could not start server on address %v!: %s\n", addr, err)
		}
		portFix = PORTFIX
		addr = fixAddress(addr)
		log.Printf("(Warning) Failed to listen on standard port, go to %v\n", addr)
		err = addr.listenAndServe(smux)
	}
	if err != nil {
		log.Fatalf("Could not start!: %s", err)
	}
}

// webCA will start the WebApp once it has been configured properly on a NEW http.ServeMux
func webCA() {
	smux := http.NewServeMux()
	addr := PrepareServer(smux)
	log.Printf("Go to %v\n", addr)
	err := addr.listenAndServe(smux)
	if err != nil {
		log.Fatalf("Could not start!: %s", err)
	}
}

// fixAddress returns a repaired alternate address by portFix
func fixAddress(a address) address {
	port := 80
	if strings.Contains(a.addr, ":") {
		parts := strings.Split(a.addr, ":")
		a.addr = parts[0]
		var err error
		port, err = strconv.Atoi(parts[1])
		if err != nil {
			port = 80
		}
	}
	a.addr = fmt.Sprintf("%s:%v", a.addr, port+portFix)
	return a
}

// prepareServer prepares the Web handlers for the setup wizard if there is no HTTPS config or 
// the normal app if the app is already configured
func PrepareServer(smux *http.ServeMux) address {
	// load config...
	cfg := LoadConfig()
	if cfg == nil { // if config is null then run the setup
		return PrepareSetup(smux) // always on the default serve mux
	}
	// otherwise start the normal app
	log.Printf("Starting WebCA normal startup...")
	smux.HandleFunc("/", index)
	smux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	return address{webCAURL(cfg), cfg.certFile(), cfg.keyFile(), true}
}

// webCAURL returns the WebCA URL
func webCAURL(cfg Configurer) string {
	certName := cfg.webCert().Crt.Subject.CommonName
	return fmt.Sprintf("%s:%v", certName, PORT+portFix)
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

// index displays the index page 
func index(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index"+_HTML, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
