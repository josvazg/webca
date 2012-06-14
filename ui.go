package webca

import (
	"bytes"
	"code.google.com/p/gorilla/context"
	"code.google.com/p/gorilla/sessions"
	"crypto/x509/pkix"
	"fmt"
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
	USERSETUP    = "userSetup"
	CASETUP      = "CASetup"
	_HTML        = ".html"
)

func init() {
	initTemplates()
}

// store for web sessions
var store = sessions.NewCookieStore([]byte("34534askjdfhkjsd41234rrf34856"))

// templates contains all web templates
var templates *template.Template

// templateIndex contains a quick way to test page template existence
var templateIndex map[string]*template.Template

// defaultHandler points to the handler for '/' requests
var defaultHandler func(w http.ResponseWriter, r *http.Request)

// CertSetaup contains the config to generate a certificate
type CertSetup struct {
	Name     pkix.Name
	Duration int
}

// SetupWizard contains the status of the setup wizard
type SetupWizard struct {
	Step   int
	CA     CertSetup
	Cert   CertSetup
	M      Mailer
	Server string
	Port   string
	Error  string
}

// PageStatus contains all values that a page and its templates need (including the SetupWizard)
type PageStatus struct {
	SetupWizard
	U     User
	Error string
}

// DisplayCertOps generates the Cert common form fields for the CA or the Cert
func (ps *PageStatus) DisplayCertOps(crt *CertSetup) template.HTML {
	ops := bytes.NewBufferString("")
	fields := []string{"StreetAddress", "PostalCode", "Locality", "Province",
		"OrganizationalUnit", "Organization", "Country"}
	labels := []string{"Street Address", "Postal Code", "Locality", "Province",
		"Organizational Unit", "Organization", "Country"}
	fieldValues := [][]string{crt.Name.StreetAddress, crt.Name.PostalCode, crt.Name.Locality,
		crt.Name.Province, crt.Name.OrganizationalUnit, crt.Name.Organization, crt.Name.Country}
	hide := ""
	prfx := "ca"
	if crt == &ps.Cert {
		hide = "style='display: none;'"
		prfx = "cert"
	}
	for i, field := range fields {
		fmt.Fprintf(ops, "<tr class='ops' %s>\n", hide)
		fmt.Fprintf(ops, "<td class='label'>%s:</td>\n", tr(labels[i]))
		fmt.Fprintf(ops, "<td><input type='text' id='%s.%s' name='%s.%s'\n",
			prfx, field, prfx, field)
		fmt.Fprintf(ops, "            value='%s'></td></tr>\n", indexOf(fieldValues[i], 0))
	}
	fmt.Fprintf(ops, "<tr class='ops' %s><td class='label'>%s:</td>\n",
		hide, tr("Duration in Days"))
	fmt.Fprintf(ops, "<td><select id='%s.Duration' name='%s.Duration' %s>\n", prfx, prfx)
	durations := []int{30, 60, 90, 180, 365, 730, 1095, 1826, 3826}
	durationLabels := []string{"1 Month", "2 Months", "3 Months", "6 Months",
		"1 Year", "2 Years", "3 Years", "5 Years", "10 Years"}
	for i, label := range durationLabels {
		sel := ""
		fmt.Fprintf(ops, "  <option value='%v' %s>%v</option>\n", durations[i], sel, tr(label))
	}
	ops.WriteString("</select></tr>\n")
	return template.HTML(ops.String())
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

// initTemplates initializes the web templates
func initTemplates() {
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

// startSetup starts the setup wizard web page sequence
func startSetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{SetupWizard: SetupWizard{Step: 1, Server: "smtp.gmail.com", Port: "587"}}
	forwardTo(w, r, ps, "setup")
}

// userSetup saves a new user
func userSetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{}
	ps.Step = 1
	ps.U.Username = r.FormValue("Username")
	ps.U.Fullname = r.FormValue("Fullname")
	ps.U.Email = r.FormValue("Email")
	pwd := r.FormValue("Password")
	pwd2 := r.FormValue("Password2")
	if pwd == "" {
		ps.Error = tr("Password is empty!")
		forwardTo(w, r, ps, "user")
		return
	} else if pwd != pwd2 {
		ps.Error = tr("Passwords don't match!")
		forwardTo(w, r, ps, "user")
		return
	}
	ps.U.Password = pwd
	ps.Step += 1
	log.Println(ps.U)
	saveInSession(w, r, USERSETUP, &ps.U)
	forwardTo(w, r, ps, "ca")
}

// userSetup saves a new user
func caSetup(w http.ResponseWriter, r *http.Request) {
	ps := &PageStatus{}
	prepareName(&ps.CA.Name)
	ps.Step = 2
	if errMsg := copyCertSetup(&ps.CA, r); errMsg != "" {
		ps.Error = errMsg
		forwardTo(w, r, ps, "ca")
		return
	}
	ps.Step += 1
	log.Println(ps.CA)
	saveInSession(w, r, CASETUP, &ps.CA)
	forwardTo(w, r, ps, "cert")
}

// copyCertSetup copies the certificate setup from the Requests Form
func copyCertSetup(cs *CertSetup, r *http.Request) string {
	cs.Name.CommonName = r.FormValue("CommonName")
	cs.Name.StreetAddress[0] = r.FormValue("StreetAddress")
	cs.Name.PostalCode[0] = r.FormValue("PostalCode")
	cs.Name.Locality[0] = r.FormValue("Locality")
	cs.Name.Province[0] = r.FormValue("Province")
	cs.Name.OrganizationalUnit[0] = r.FormValue("OrganizationalUnit")
	cs.Name.Organization[0] = r.FormValue("Organization")
	cs.Name.Country[0] = r.FormValue("Country")
	duration, err := strconv.Atoi(r.FormValue("Duration"))
	if err != nil || duration < 0 {
		return tr("Wrong duration!")
	}
	cs.Duration = duration
	return ""
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
		forwardTo(w, r, ps, "mailer")
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

// saveInSession saves the key-value pair in the session
func saveInSession(w http.ResponseWriter, r *http.Request, key string, value interface{}) {
	session, _ := store.Get(r, "")
	session.Values[key] = value
	session.Save(r, w)
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
	log.Println("uri=", r.URL.RequestURI(), "page=", page, "ps=", ps)
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
	smux.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	smux.HandleFunc("/userSetup", userSetup)
	smux.HandleFunc("/caSetup", caSetup)
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
func RegisterSetup() {
	http.HandleFunc("/", autoPage)
	http.Handle("/img/", http.StripPrefix("/img/", http.FileServer(http.Dir("img"))))
	http.HandleFunc("/userSetup", userSetup)
	http.HandleFunc("/caSetup", caSetup)
}

/*
func main() {
	webca()
}*/

