package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
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

type CertPair struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
}

type CertNode struct {
	ca    *CertPair
	certs []*CertPair
	next  *CertNode
}

type CertTree struct {
	caNode map[string]*CertNode
	first  *CertNode
	last   *CertNode
}

func GenCACert(name pkix.Name, years int) *CertPair {
	return genCert(nil, name, years)
}

func GenCert(parent *CertPair, cert string, years int) *CertPair {
	name := copyName(parent.cert.Subject)
	name.CommonName = cert
	return genCert(parent, name, years)
}

func RenewCert(parent *CertPair, cp *CertPair) *CertPair {
	years := int((cp.cert.NotAfter.Unix() - cp.cert.NotBefore.Unix()) / SECS_IN_YEAR)
	return genCert(parent, cp.cert.Subject, years)
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

func genCert(p *CertPair, name pkix.Name, years int) *CertPair {
	tmpl := &CertPair{}
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
		return nil
	}
	tmpl.key = key
	now := time.Now()
	tmpl.cert = &x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject:      name,
		NotBefore:    now.Add(-5 * time.Minute).UTC(),
		NotAfter:     now.AddDate(years, 0, 0).UTC(), // valid for years

		SubjectKeyId: []byte{1, 2, 3, 4},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	if p == nil {
		tmpl.cert.BasicConstraintsValid = true
		tmpl.cert.IsCA = true
		tmpl.cert.MaxPathLen = 0
		p = tmpl
	}

	certname := name.CommonName + CERT_SUFFIX
	keyname := name.CommonName + KEY_SUFFIX

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl.cert, p.cert, &p.key.PublicKey, p.key)
	//log.Println("Generated:", tmpl)
	if err != nil {
		log.Fatalf("Failed to create Certificate: %s", err)
		return nil
	}

	certOut, err := os.Create(certname)
	if err != nil {
		log.Fatalf("Failed to open "+certname+" for writing: %s", err)
		return nil
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()
	//log.Print("Written " + certname + "\n")

	keyOut, err := os.OpenFile(keyname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("Failed to open "+keyname+" for writing:", err)
		return nil
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(tmpl.key)})
	keyOut.Close()
	//log.Print("Written " + keyname + "\n")
	return tmpl
}

func loadCertPair(name string) *CertPair {
	cp := CertPair{}
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
		log.Fatalf("Failed to open "+name+" for reading: %s", err)
		return nil
	}
	b, _ := pem.Decode(certIn)
	if b == nil {
		log.Fatalf("Failed to find a certificate in " + name)
		return nil
	}
	cp.cert, err = x509.ParseCertificate(b.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse certificate " + name)
		return nil
	}
	keyIn, err := ioutil.ReadFile(kname)
	if err != nil {
		log.Fatalf("Failed to open "+kname+" for reading: %s", err)
		return nil
	}
	kb, _ := pem.Decode(keyIn)
	if kb == nil {
		log.Fatalf("Failed to find a key in " + kname)
		return nil
	}
	cp.key, err = x509.ParsePKCS1PrivateKey(kb.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse key " + kname)
		return nil
	}
	return &cp
}

func LoadCertTree(dir string) *CertTree {
	ctree := &CertTree{make(map[string]*CertNode, 0), nil, nil}
	fi, err := os.Lstat(dir)
	if err != nil {
		log.Fatalf("Failed to check path "+dir+":", err)
		return nil
	}
	if !fi.IsDir() {
		log.Fatalf("Path " + dir + " is not a directory!")
		return nil
	}
	f, err := os.Open(dir)
	if err != nil {
		log.Fatalf("Can't open "+dir+" for reading:", err)
		return nil
	}
	defer f.Close()
	for fis, err := f.Readdir(100); err == nil; fis, err = f.Readdir(100) {
		for _, fi := range fis {
			if !fi.IsDir() && strings.HasSuffix(fi.Name(), CERT_SUFFIX) &&
				!strings.HasSuffix(fi.Name(), KEY_SUFFIX) {
				ctree.addCert(loadCertPair(fi.Name()))
			}
		}
	}
	if err != nil {
		log.Fatalf("Can't read dir "+dir+":", err)
		return nil
	}
	if len(ctree.caNode) == 0 {
		return nil
	}
	return ctree
}

func (ct *CertTree) addCA(cp *CertPair) {
	cnode, _ := ct.caNode[cp.cert.Subject.CommonName]
	if cnode == nil {
		ct.addCertNode(cp.cert.Subject.CommonName, &CertNode{cp, make([]*CertPair, 0), nil})
	} else {
		cnode.ca = cp
	}
}

func (ct *CertTree) addCert(cp *CertPair) {
	if cp.cert.IsCA {
		ct.addCA(cp)
		return
	}
	cnode, _ := ct.caNode[cp.cert.Issuer.CommonName]
	if cnode == nil {
		ct.addCertNode(cp.cert.Issuer.CommonName, &CertNode{nil, []*CertPair{cp}, nil})
	} else {
		cnode.certs = append(cnode.certs, cp)
	}
}

func (ct *CertTree) addCertNode(name string, cnode *CertNode) {
	ct.caNode[name] = cnode
	if ct.first == nil || ct.last == nil {
		ct.first, ct.last = cnode, cnode
	} else {
		ct.last.next = cnode
		ct.last = cnode
	}
}

func (ct *CertTree) String() string {
	s := ""
	if ct.first != nil {
		for cn := ct.first; cn != nil; cn = cn.next {
			s += cn.String()
		}
	}
	return s
}

func (cn *CertNode) String() string {
	s := cn.ca.String() + ":\n"
	for _, crt := range cn.certs {
		s += "    " + crt.String() + "\n"
	}
	if cn.next == nil {
		s += "END\n"
	}
	return s
}

func (cp *CertPair) String() string {
	prefix := ""
	if cp.cert.IsCA {
		prefix = "(CA)"
	}
	return "CertPair " + prefix + " " + cp.cert.Subject.CommonName +
		" (" + cp.cert.NotBefore.String() + " - " + cp.cert.NotAfter.String() + ")"
}

func main() {
	certTree := LoadCertTree(".")
	if certTree == nil {
		ca := GenCACert(pkix.Name{CommonName: "TestCA",
			StreetAddress:      []string{"Acme st. num. 23"},
			PostalCode:         []string{"12345"},
			Locality:           []string{"Acme City"},
			Province:           []string{"Acme County"},
			OrganizationalUnit: []string{"Acme Labs"},
			Organization:       []string{"Acme"},
			Country:            []string{"AcmeLand"}}, 4)
		GenCert(ca, "server.acme.com", 2)
		certTree = LoadCertTree(".")
	}
	log.Print("CertTree:\n", certTree)

	log.Print("CertTree.first:\n", certTree.first)
	RenewCert(nil, certTree.first.ca)
	RenewCert(certTree.first.ca, certTree.first.certs[0])
	certTree = LoadCertTree(".")
	log.Print("Renewed CertTree:\n", certTree)
}

