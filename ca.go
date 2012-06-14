package webca

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	//"errors"
	"fmt"
	//"flag"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

const (
	CERT_SUFFIX  = ".pem"
	KEY_SUFFIX   = ".key.pem"
	SECS_IN_YEAR = 365 * 24 * 60 * 60
)

type Cert struct {
	crt    *x509.Certificate
	key    *rsa.PrivateKey
	parent *Cert // parent (CA) cert if any
}

type CertTree struct {
	certs  map[string]*Cert   // Name to cert mapper
	cakids map[string][]*Cert // CA to cert list
	order  []*Cert            // Ordered list of certs
}

func GenCACert(name pkix.Name, years int) (*Cert, error) {
	return genCert(nil, name, years)
}

func GenCert(parent *Cert, cert string, years int) (*Cert, error) {
	name := copyName(parent.crt.Subject)
	name.CommonName = cert
	return genCert(parent, name, years)
}

func RenewCert(cert *Cert) (*Cert, error) {
	years := int((cert.crt.NotAfter.Unix() - cert.crt.NotBefore.Unix()) / SECS_IN_YEAR)
	return genCert(cert.parent, cert.crt.Subject, years)
}

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

func genCert(p *Cert, name pkix.Name, years int) (*Cert, error) {
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
	t.crt = &x509.Certificate{
		SerialNumber: serial,
		Subject:      name,
		NotBefore:    now.Add(-5 * time.Minute).UTC(),
		NotAfter:     now.AddDate(years, 0, 0).UTC(), // valid for years

		SubjectKeyId: ski,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	t.key = key
	if p == nil {
		t.crt.BasicConstraintsValid = true
		t.crt.IsCA = true
		t.crt.MaxPathLen = 0
		t.crt.KeyUsage = t.crt.KeyUsage | x509.KeyUsageCertSign
		p = t
		//log.Println("t.key.PublicKey=", t.key.PublicKey)
		//log.Println("p.key=", t.key)
	} else {
		t.parent = p
	}

	certname := name.CommonName + CERT_SUFFIX
	keyname := name.CommonName + KEY_SUFFIX

	derBytes, err := x509.CreateCertificate(rand.Reader, t.crt, p.crt, &t.key.PublicKey, p.key)
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
		Bytes: x509.MarshalPKCS1PrivateKey(t.key)})
	keyOut.Close()
	//log.Print("Written " + keyname + "\n")
	return t, nil
}

func loadCert(name string) (*Cert, error) {
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
	cert.crt, err = x509.ParseCertificate(b.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse certificate " + name)
	}
	keyIn, err := ioutil.ReadFile(kname)
	if err != nil {
		return nil, fmt.Errorf("Failed to open "+kname+" for reading: %s", err)
	}
	kb, _ := pem.Decode(keyIn)
	if kb == nil {
		return nil, fmt.Errorf("Failed to find a key in " + kname)
	}
	cert.key, err = x509.ParsePKCS1PrivateKey(kb.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key " + kname)
	}
	return &cert, nil
}

func LoadCertTree(dir string) *CertTree {
	ctree := newCertTree()
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
				if crt, err := loadCert(fi.Name()); err == nil {
					ctree.addCert(crt)
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
	if len(ctree.certs) == 0 {
		return nil
	}
	return ctree
}

func newCertTree() *CertTree {
	return &CertTree{make(map[string]*Cert, 0), make(map[string][]*Cert, 0), make([]*Cert, 0)}
}

func (ct *CertTree) insertCert(cert *Cert) {
	ct.certs[cert.crt.Subject.CommonName] = cert
	ct.order = append(ct.order, cert)
	if cert.crt.IsCA {
		kids := ct.cakids[cert.crt.Subject.CommonName]
		if kids == nil {
			kids = make([]*Cert, 0)
			ct.cakids[cert.crt.Subject.CommonName] = kids
		} else {
			for _, kidcrt := range kids {
				kidcrt.parent = cert
			}
		}
	} else {
		kids := ct.cakids[cert.crt.Issuer.CommonName]
		if kids == nil {
			kids = []*Cert{cert}
			ct.cakids[cert.crt.Issuer.CommonName] = kids
		} else {
			ct.cakids[cert.crt.Issuer.CommonName] = append(kids, cert)
		}
		cacert := ct.certs[cert.crt.Issuer.CommonName]
		if cacert != nil {
			cert.parent = cacert
		}
	}
}

func (ct *CertTree) addCert(cert *Cert) {
	prev := ct.certs[cert.crt.Subject.CommonName]
	if prev == nil {
		ct.insertCert(cert)
		return
	}
	if cert.crt != nil {
		prev.crt = cert.crt
	}
	if cert.parent != nil {
		prev.parent = cert.parent
	}
	if cert.key != nil {
		prev.key = cert.key
	}
}

func (ct *CertTree) String() string {
	s := "CertTree:\n"
	for ca, kids := range ct.cakids {
		s += ct.certs[ca].String() + "\n"
		for _, crt := range kids {
			s += "    " + crt.String() + "\n"
		}
	}
	return s
}

func (cert *Cert) String() string {
	prefix := ""
	if cert.crt.IsCA {
		prefix = "(CA)"
	}
	return prefix + " " + cert.crt.Subject.CommonName +
		" (" + cert.crt.NotBefore.String() + " - " + cert.crt.NotAfter.String() + ")"
}

func handleFatal(err error) {
	if err != nil {
		log.Fatalf(err.Error())
	}
}

/*
func main() {
	certTree := LoadCertTree(".")
	if certTree == nil {
		certTree = newCertTree()
		ca, err := GenCACert(pkix.Name{CommonName: "TestCA",
			StreetAddress:      []string{"Acme st. num. 23"},
			PostalCode:         []string{"12345"},
			Locality:           []string{"Acme City"},
			Province:           []string{"Acme County"},
			OrganizationalUnit: []string{"Acme Labs"},
			Organization:       []string{"Acme"},
			Country:            []string{"AcmeLand"}}, 4)
		handleFatal(err)
		certTree.addCert(ca)
		for _, crtName := range []string{"server.acme.com", "tys14ubu.rfranco.com"} {
			crt, err := GenCert(ca, crtName, 2)
			handleFatal(err)
			certTree.addCert(crt)
		}
	}
	log.Print(certTree)

	//log.Print("CertTree.first:\n", certTree.first)
	//RenewCert(nil, certTree.first.ca)
	//RenewCert(certTree.first.ca, certTree.first.certs[0])
	//certTree = LoadCertTree(".")
	//log.Print("Renewed CertTree:\n", certTree)
}*/

