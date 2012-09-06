package webca

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	PORT       = 443
	PORTFIX    = 8000
	LOGGEDUSER = "LoggedUser"
)

// address is a complex bind address
type address struct {
	addr, certfile, keyfile string
	tls                     bool
}

// fakedLogin for development environments
var fakedLogin bool

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
		"tr": tr, "indexOf": indexOf, "showPeriod": showPeriod,
	})
	template.Must(templates.Parse(htmlTemplates))
	template.Must(templates.Parse(jsTemplates))
	template.Must(templates.Parse(pages))
	template.Must(templates.ParseFiles("style.css"))
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

func (ps PageStatus) Url(path string, args ... string) string {
	buf:=bytes.NewBufferString(path+"?")
	fmt.Println("ps[SESSIONID]=",ps[SESSIONID])
	buf.WriteString(url.QueryEscape(SESSIONID)+"=")
	join:="&"
	for n,arg := range args {
		s:=join
		if (n%2)!=0 {
			s="="
		}
		buf.WriteString(s+url.QueryEscape(arg))
	}
	return buf.String()
}

// tr is the app translation function
func tr(s string, args ...interface{}) string {
	if args == nil || len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
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
	if cfg == nil { // if config is empty then run the setup
		return PrepareSetup(smux) // always on the default serve mux
	}
	// otherwise start the normal app
	log.Printf("Starting WebCA normal startup...")
	smux.Handle("/", accessControl(index))
	smux.HandleFunc("/login", login)
	smux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	smux.Handle("/favicon.ico", http.FileServer(http.Dir("img")))
	return address{webCAURL(cfg), certFile(cfg.getWebCert()), keyFile(cfg.getWebCert()), true}
}

// webCAURL returns the WebCA URL
func webCAURL(cfg *config) string {
	certName := cfg.getWebCert().Crt.Subject.CommonName
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
	ps := copyRequest(PageStatus{}, r)
	ct := LoadCertree(".")
	ps["CAs"] = ct.roots
	ps["Others"] = ct.foreign
	fmt.Println("Url=",ps.Url("edit","cert","name"))
	err := templates.ExecuteTemplate(w, "index", ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// accessControl invokes handler h ONLY IF we are logged in, otherwise the login page
func accessControl(h func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, err := SessionFor(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		if s[LOGGEDUSER] == nil {
			if fakedLogin {
				s[LOGGEDUSER] = User{"fuser", "Faked User", "****", "fuser@fuser.com"}
				s.Save()
				redirectWithSession(s, w, r)
				return
			}
			ps := PageStatus{}
			ps["URL"] = r.URL
			ps[SESSIONID] = s[SESSIONID].(string)
			err := templates.ExecuteTemplate(w, "login", copyRequest(ps, r))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		h(w, r)
	})
}

// login handles login action
func login(w http.ResponseWriter, r *http.Request) {
	Username := r.FormValue("Username")
	Password := crypt(r.FormValue("Password"))
	cfg := LoadConfig()
	u := cfg.getUser(Username)
	if u.Password != Password {
		ps := copyRequest(PageStatus{"Error": tr("Access Denied")}, r)
		err := templates.ExecuteTemplate(w, "login", ps)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	} else {
		s, err := SessionFor(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		s[LOGGEDUSER] = u
		s.Save()
		redirectWithSession(s, w, r)
	}
}

// redirectWithSession redirects to the same URL but adding the session request parameter
func redirectWithSession(s session, w http.ResponseWriter, r *http.Request) {
	targetUrl := r.FormValue("URL")
	if targetUrl == "" {
		targetUrl = "/"
	}
	targetUrl, err := addPar(targetUrl, SESSIONID, s[SESSIONID].(string))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	http.Redirect(w, r, targetUrl, 302)
}

// copyRequest copies a request status data on a PageStatus
func copyRequest(ps PageStatus, r *http.Request) PageStatus {
	r.ParseForm()
	s, err := SessionFor(r)
	if err == nil {
		ps["Session"] = s
	}
	for k, v := range r.Form {
		if v != nil {
			if len(v) == 1 {
				ps[k] = v[0]
			} else {
				ps[k] = v
			}
		}
	}
	return ps
}

// addPar adds or resets a parameter id to the given Url
func addPar(aUrl, key, value string) (string, error) {
	newUrl, err := url.Parse(aUrl)
	if err != nil {
		return "", err
	}
	values := newUrl.Query()
	values.Del(key)
	values.Add(key, value)
	newUrl.RawQuery = ""
	return newUrl.RequestURI() + "?" + values.Encode(), nil
}

// fakeLogin fakes the login process
func FakeLogin() {
	fakedLogin = true
}
