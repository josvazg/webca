package webca

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	CERT_SUFFIX = ".pem"
	KEY_SUFFIX  = ".key.pem"
	MYFMT       = "2006/01/02"
)

// Cert holds the certificate the key and links to parent and children
type Cert struct {
	Crt    *x509.Certificate
	Key    *rsa.PrivateKey
	Parent *Cert   // parent (CA) cert if any
	Childs []*Cert // children (CA) certs if any
}

// Certree holds a certificate tree
type Certree struct {
	names   map[string]*Cert
	roots   []*Cert
	foreign []*Cert
}

// certree in memory
var certree *Certree

// certree access lock
var scerts sync.RWMutex

// GenCACert generates a CA Certificate, that is a self signed certificate
func GenCACert(name pkix.Name, days int) (*Cert, error) {
	cert, err := genCert(nil, name, days)
	if err != nil {
		return nil, err
	}
	certree = nil // forces full reload later
	return cert, nil
}

// CenCert generates a Certificate signed by another certificate
func GenCert(parent *Cert, certname string, days int) (*Cert, error) {
	name := copyName(parent.Crt.Subject)
	name.CommonName = certname
	cert, err := genCert(parent, name, days)
	if err != nil {
		return nil, err
	}
	certree = nil // forces full reload later
	return cert, nil
}

// RenewCert renews the given certificate for the same duration as before from now
func RenewCert(cert *Cert) (*Cert, error) {
	days := int(cert.Crt.NotAfter.Sub(cert.Crt.NotBefore).Hours() / 24)
	cert, err := genCert(cert.Parent, cert.Crt.Subject, days)
	if err != nil {
		return nil, err
	}
	certree = nil // forces full reload later
	return cert, nil
}

// ListCerts returns the current Certree
func ListCerts() *Certree {
	return autoload()
}

// FindCert finds a certificate by name
func FindCert(certname string) *Cert {
	autoload()
	scerts.RLock()
	defer scerts.RUnlock()
	return certree.names[certname]
}

// ReadCert reads the Certificate Contents
func ReadCert(cert *Cert) ([]byte, error) {
	return ioutil.ReadFile(certFile(*cert))
}

// ReadCertKey reads the Certificate Key contents
func ReadCertKey(cert *Cert) ([]byte, error) {
	return ioutil.ReadFile(keyFile(*cert))
}

// CloneCert generates a clone of the original certificate with a new name
func CloneCert(cert *Cert, newname string) *Cert {
	c := &Cert{Crt: &x509.Certificate{Subject: copyName(cert.Crt.Subject)}, Parent: cert.Parent}
	c.Crt.Subject.CommonName = newname
	return c
}

// DeleteCert deletes a certificate
func DeleteCert(cert *Cert) bool {
	scerts.Lock()
	defer scerts.Unlock()
	if err := os.Remove(certFile(*cert)); err != nil {
		return false
	}
	if err := os.Remove(keyFile(*cert)); err != nil {
		return false
	}
	certree = nil // forces full reload later
	return true
}

// autoload will autoload certree
func autoload() *Certree {
	scerts.Lock()
	defer scerts.Unlock()
	if certree == nil {
		certree = loadCertree(".")
	}
	return certree
}

// copyName generates a copy of the given Certificate name
func copyName(name pkix.Name) pkix.Name {
	return pkix.Name{CommonName: name.CommonName,
		StreetAddress:      name.StreetAddress,
		PostalCode:         name.PostalCode,
		Locality:           name.Locality,
		Province:           name.Province,
		OrganizationalUnit: name.OrganizationalUnit,
		Organization:       name.Organization,
		Country:            name.Country,
	}
}

// Prepare name clears a Certificate name
func prepareName(name *pkix.Name) {
	if name.Country == nil {
		name.StreetAddress = []string{""}
		name.PostalCode = []string{""}
		name.Province = []string{""}
		name.Locality = []string{""}
		name.OrganizationalUnit = []string{""}
		name.Organization = []string{""}
		name.Country = []string{""}
	}
}

// genCert generates a certificated signed by itself or by another certificate
func genCert(p *Cert, name pkix.Name, days int) (*Cert, error) {
	t := &Cert{}
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate private key: %s", err)
	}
	now := time.Now()
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(9223372036854775807))
	ski := []byte{0, 0, 0, 0}
	rand.Reader.Read(ski)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate random serial number: %s", err)
	}
	//log.Println("serial:", serial)
	//log.Println("ski:", ski)
	t.Crt = &x509.Certificate{
		SerialNumber: serial,
		Subject:      name,
		NotBefore:    now.Add(-5 * time.Minute).UTC(),
		NotAfter:     now.AddDate(0, 0, days).UTC(), // valid for days

		SubjectKeyId: ski,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	t.Key = key
	if p == nil {
		t.Crt.BasicConstraintsValid = true
		t.Crt.IsCA = true
		t.Crt.MaxPathLen = 0
		t.Crt.KeyUsage = t.Crt.KeyUsage | x509.KeyUsageCertSign
		p = t
		//log.Println("t.Key.PublicKey=", t.Key.PublicKey)
		//log.Println("p.Key=", t.Key)
	} else {
		t.Parent = p
	}

	certname := name.CommonName + CERT_SUFFIX
	keyname := name.CommonName + KEY_SUFFIX

	derBytes, err := x509.CreateCertificate(rand.Reader, t.Crt, p.Crt, &t.Key.PublicKey, p.Key)
	//log.Println("Generated:", tmpl)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Certificate: %s", err)
	}

	certOut, err := os.Create(certname)
	if err != nil {
		return nil, fmt.Errorf("Failed to open "+certname+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	//log.Print("Written " + certname + "\n")

	keyOut, err := os.OpenFile(keyname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("Failed to open "+keyname+" for writing: %s", err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(t.Key)})
	keyOut.Close()
	//log.Print("Written " + keyname + "\n")
	return t, nil
}

// readCert loads a Cert and Key pair from disk .pem files
func readCert(name string) (*Cert, error) {
	cert := Cert{}
	kname := name
	if strings.HasSuffix(kname, CERT_SUFFIX) {
		kname = kname[:len(kname)-len(CERT_SUFFIX)]
	}
	if !strings.HasSuffix(kname, KEY_SUFFIX) {
		kname = kname + KEY_SUFFIX
	}
	if !strings.HasSuffix(name, CERT_SUFFIX) {
		name = name + CERT_SUFFIX
	}
	certIn, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("Failed to open "+name+" for reading: %s", err)
	}
	b, _ := pem.Decode(certIn)
	if b == nil {
		return nil, fmt.Errorf("Failed to find a certificate in " + name)
	}
	cert.Crt, err = x509.ParseCertificate(b.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse certificate " + name)
	}
	_, err = os.Stat(kname)
	if os.IsNotExist(err) {
		return &cert, nil
	}
	keyIn, err := ioutil.ReadFile(kname)
	if err != nil {
		return nil, fmt.Errorf("Failed to open "+kname+" for reading: %s", err)
	}
	kb, _ := pem.Decode(keyIn)
	if kb == nil {
		return nil, fmt.Errorf("Failed to find a key in " + kname)
	}
	cert.Key, err = x509.ParsePKCS1PrivateKey(kb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key " + kname)
	}
	return &cert, nil
}

// NewCertree generates an empty Certree
func newCertree() *Certree {
	return &Certree{make(map[string]*Cert), make([]*Cert, 0), make([]*Cert, 0)}
}

// loadCertree will load all found .pem certs and keys on a Certree
func loadCertree(dir string) *Certree {
	ct := newCertree()
	fi, err := os.Lstat(dir)
	if err != nil {
		log.Printf("(Warning) Failed to check path "+dir+":", err)
		return nil
	}
	if !fi.IsDir() {
		log.Printf("(Warning) Path " + dir + " is not a directory!")
		return nil
	}
	f, err := os.Open(dir)
	if err != nil {
		log.Printf("(Warning) Can't open "+dir+" for reading:", err)
		return nil
	}
	defer f.Close()
	for fis, err := f.Readdir(100); err == nil; fis, err = f.Readdir(100) {
		for _, fi := range fis {
			if !fi.IsDir() && strings.HasSuffix(fi.Name(), CERT_SUFFIX) &&
				!strings.HasSuffix(fi.Name(), KEY_SUFFIX) {
				if crt, err := readCert(fi.Name()); err == nil {
					ct.add(crt)
				} else {
					log.Printf("(Warning) %s", err)
				}
			}
		}
	}
	if err != nil {
		log.Printf("(Warning) Can't read dir "+dir+":", err)
		return nil
	}
	if len(ct.roots) == 0 && len(ct.foreign) == 0 {
		return nil
	}
	return ct
}

// add or replace a certificate in its ordered position within the Cert list
func (ct *Certree) add(crt *Cert) {
	cn := ct.names[crt.Crt.Subject.CommonName]
	if cn == nil { // if unknown, create and register in certnames
		ct.names[crt.Crt.Subject.CommonName] = crt
		cn = crt
	} else { // update cert info otherwise
		cn.Crt = crt.Crt
		cn.Key = crt.Key
	}
	// if root just place it and we are done
	if crt.Crt.Subject.CommonName == crt.Crt.Issuer.CommonName {
		cn.Parent = cn
		if crt.Key != nil {
			ct.roots = place(ct.roots, cn)
			ct.foreign = remove(ct.foreign, cn)
		} else {
			ct.foreign = place(ct.foreign, cn)
		}
		return
	} else { // otherwise we must find the parent and link the kid
		parent := ct.names[crt.Crt.Issuer.CommonName]
		if parent == nil { // if parent is unknown, generate a Cert for it and register
			parent = &Cert{Crt: &x509.Certificate{Subject: copyName(crt.Crt.Issuer)},
				Childs: make([]*Cert, 0),
			}
			ct.names[crt.Crt.Issuer.CommonName] = parent
		}
		cn.Parent = parent
		parent.Childs = place(parent.Childs, cn)
	}
	// is this cert part of a known hierarchy or a loose end?
	current := cn
	for current.Parent != nil && current.Crt.Issuer.CommonName != current.Crt.Subject.CommonName {
		current = current.Parent
	}
	if current.Parent == nil { // loose end goes to rest
		ct.foreign = place(ct.foreign, current)
	}
}

// place kid in order under the given childs list and returns the new ordered and appended list
func place(childs []*Cert, kid *Cert) []*Cert {
	candidate := kid
	for i, _ := range childs {
		if candidate.Crt.Subject.CommonName == childs[i].Crt.Subject.CommonName { // already there
			return childs
		}
		if candidate.Crt.Subject.CommonName < childs[i].Crt.Subject.CommonName {
			candidate, childs[i] = childs[i], candidate
		}
	}
	return append(childs, candidate)
}

// remove will return a childs list where there is no kid
func remove(childs []*Cert, kid *Cert) []*Cert {
	for _, child := range kid.Childs {
		childs = remove(childs, child)
	}
	candidate := kid
	for i, _ := range childs {
		// if found, remove
		if candidate.Crt.Subject.CommonName == childs[i].Crt.Subject.CommonName {
			for j := i; j < len(childs)-1; j++ {
				childs[j] = childs[j+1]
			}
			return childs[:len(childs)-1]
		}
	}
	return childs // not found, we return the same list
}

// String will return the recursive string representation for a Cert
func (c *Cert) String() string {
	return printCert(c, "  ")
}

// printCert returns the recursive string representation for a Cert with a given left margin
func printCert(c *Cert, margin string) string {
	str := bytes.NewBufferString(margin)
	if c.Crt.IsCA {
		str.WriteString("(CA)")
	}
	str.WriteString(c.Crt.Subject.CommonName)
	str.WriteString(" (" + c.Crt.NotBefore.Format(MYFMT) + " - " +
		c.Crt.NotAfter.Format(MYFMT) + ")\n ")
	for _, k := range c.Childs {
		str.WriteString(printCert(k, margin+"  "))
	}
	return str.String()
}

// String will return the string representation for a Certree
func (ct *Certree) String() string {
	str := bytes.NewBufferString("-- ROOTS --\n")
	for _, k := range ct.roots {
		str.WriteString(k.String())
	}
	str.WriteString("-- FOREIGN --\n")
	for _, k := range ct.foreign {
		str.WriteString(k.String())
	}
	return str.String()
}

// handleFatal will show the fatal error and exit inmediatelly
func handleFatal(err error) {
	if err != nil {
		log.Fatalf(err.Error())
	}
}

// certFile returns the cert filename for a given Certificate
func certFile(crt Cert) string {
	return filename(crt.Crt.Subject.CommonName) + CERT_SUFFIX
}

// keyFile returns the key filename for a given Certificate
func keyFile(crt Cert) string {
	return filename(crt.Crt.Subject.CommonName) + KEY_SUFFIX
}

// filename filters a name to make sure is a legal filename
func filename(name string) string {
	return name // TODO ensure result is a proper filename without forbidden chars
}

// showPeriod shows the period of a Certificate
func showPeriod(crt *x509.Certificate) string {
	from := crt.NotBefore.Format(MYFMT)
	to := crt.NotAfter.Format(MYFMT)
	duration := int(crt.NotAfter.Sub(time.Now()).Hours() / 24)
	return tr("From %s to %s (%ddays to go)", from, to, duration)
}
