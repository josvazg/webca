package webca

import (
	"bytes"
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
	_HTML        = ".html"
)

func init() {
	initTemplates()
}


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

// SetupWizard contains the status of the setup wizard
//	CA     CertSetup
//	Cert   CertSetup
//	M      Mailer

// PageStatus contains all values that a page and its templates need (including the SetupWizard)
//	U     User
type PageStatus map[string]interface{}

// DisplayCertOps generates the Cert common form fields for the CA or the Cert
func (ps PageStatus) DisplayCertOps(arg interface{}) template.HTML {
	crt:=arg.(*CertSetup)
	ops := bytes.NewBufferString("")
	fields := []string{"StreetAddress", "PostalCode", "Locality", "Province",
		"OrganizationalUnit", "Organization", "Country"}
	labels := []string{"Street Address", "Postal Code", "Locality", "Province",
		"Organizational Unit", "Organization", "Country"}
	fieldValues := [][]string{crt.Name.StreetAddress, crt.Name.PostalCode, crt.Name.Locality,
		crt.Name.Province, crt.Name.OrganizationalUnit, crt.Name.Organization, crt.Name.Country}
	hide := ""
	prfx := "CA"
	duration := 1095
	if crt == (ps["Cert"]).(*CertSetup) {
		hide = "style='display: none;'"
		prfx = "Cert"
		duration = 365
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
		if durations[i] == duration {
			sel = "selected='selected'"
		}
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
	ps := PageStatus{
		"Server": "smtp.gmail.com", 
		"Port": "587",
		"CA": &CertSetup{},
		"Cert": &CertSetup{},
		"U":&User{},
		"M":&Mailer{},
	}
	err := templates.ExecuteTemplate(w, "setup"+_HTML, ps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// endSetup saves the initial setup 
func endSetup(w http.ResponseWriter, r *http.Request) {
	user:=readUser(r)
	certs:=make(map[string]*CertSetup,2)
	for _,prefix := range []string{"CA","Cert"} {
		crt,err:=readCertSetup(prefix,r)
		if err!=nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		certs[prefix]=crt
	}
	mailer:=readMailer(r)
	log.Println("user=",user)
	log.Println("certs=",certs)
	log.Println("mailer=",mailer)
}

// readUser reads the user data into PageStatus
func readUser(r *http.Request) User {
	u:=User{}
	u.Username = r.FormValue("Username")
	u.Fullname = r.FormValue("Fullname")
	u.Email = r.FormValue("Email")
	u.Password = r.FormValue("Password")
	return u
}

// readCertSetup copies the certificate setup from the Requests Form
func readCertSetup(prefix string, r *http.Request) (*CertSetup,error) {
	cs:=CertSetup{}
	prepareName(&cs.Name)
	cs.Name.CommonName = r.FormValue(prefix+".CommonName")
	cs.Name.StreetAddress[0] = r.FormValue(prefix+"StreetAddress")
	cs.Name.PostalCode[0] = r.FormValue(prefix+"PostalCode")
	cs.Name.Locality[0] = r.FormValue(prefix+"Locality")
	cs.Name.Province[0] = r.FormValue(prefix+"Province")
	cs.Name.OrganizationalUnit[0] = r.FormValue(prefix+"OrganizationalUnit")
	cs.Name.Organization[0] = r.FormValue(prefix+"Organization")
	cs.Name.Country[0] = r.FormValue(prefix+"Country")
	duration, err := strconv.Atoi(r.FormValue(prefix+"Duration"))
	if err != nil || duration < 0 {
		return nil,fmt.Errorf("%s: %v",tr("Wrong duration!"),err)
	}
	cs.Duration = duration
	return &cs,nil
}

// readMailer copies the mailer settings
func readMailer(r *http.Request) Mailer {
	m:=Mailer{}
	m.User = r.FormValue("M.User")
	m.Server = r.FormValue("M.Server")
	port := r.FormValue("M.Port")
	if port != "" {
		m.Server += ":" + port
	}
	m.Passwd = r.FormValue("M.Password")
	return m
}

// autoPage loads current pageStatus and then displays the page given in the URL
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
	smux.HandleFunc("/endSetup",endSetup)
}


