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
	CERT_SUFFIX = ".pem"
	KEY_SUFFIX  = ".key.pem"
)

type CertNode struct {
	ca    *x509.Certificate
	certs []*x509.Certificate
	next  *CertNode
}

type CertTree struct {
	caNode map[string]*CertNode
	first  *CertNode
	last   *CertNode
}

func GenCACert(name pkix.Name, years int) *x509.Certificate {
	now := time.Now()
	template := x509.Certificate{
		SerialNumber:          new(big.Int).SetInt64(0),
		Subject:               name,
		NotBefore:             now.Add(-5 * time.Minute).UTC(),
		NotAfter:              now.AddDate(years, 0, 0).UTC(), // valid for years
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,

		SubjectKeyId: []byte{1, 2, 3, 4},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	return genCert(name, years, &template, &template)
}

func GenCert(cert string, parent *x509.Certificate, years int) *x509.Certificate {
	now := time.Now()
	name := copyName(parent.Subject)
	name.CommonName = cert
	template := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject:      name,
		NotBefore:    now.Add(-5 * time.Minute).UTC(),
		NotAfter:     now.AddDate(years, 0, 0).UTC(), // valid for years

		SubjectKeyId: []byte{1, 2, 3, 4},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	return genCert(name, years, &template, parent)
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

func genCert(name pkix.Name, years int, template, parent *x509.Certificate) *x509.Certificate {
	certname := name.CommonName + CERT_SUFFIX
	keyname := name.CommonName + KEY_SUFFIX
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
		return nil
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, &priv.PublicKey, priv)
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
	log.Print("Written " + certname + "\n")

	keyOut, err := os.OpenFile(keyname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Print("Failed to open "+keyname+" for writing:", err)
		return nil
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	log.Print("Written " + keyname + "\n")
	return template
}

func loadCert(name string) *x509.Certificate {
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
	cert, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse certificate " + name)
		return nil
	}
	return cert
}

func BuildCertTree(dir string) *CertTree {
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
				ctree.addCert(loadCert(fi.Name()))
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

func (ct *CertTree) addCA(crt *x509.Certificate) {
	cnode, _ := ct.caNode[crt.Subject.CommonName]
	if cnode == nil {
		ct.addCertNode(crt.Subject.CommonName, &CertNode{crt, make([]*x509.Certificate, 0), nil})
	} else {
		cnode.ca = crt
	}
}

func (ct *CertTree) addCert(crt *x509.Certificate) {
	if crt.IsCA {
		ct.addCA(crt)
		return
	}
	cnode, _ := ct.caNode[crt.Issuer.CommonName]
	if cnode == nil {
		ct.addCertNode(crt.Issuer.CommonName, &CertNode{nil, []*x509.Certificate{crt}, nil})
	} else {
		cnode.certs = append(cnode.certs, crt)
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
	s := "CA " + cn.ca.Subject.CommonName + ":\n"
	for _, crt := range cn.certs {
		s += "    " + crt.Subject.CommonName + "\n"
	}
	return s
}

func main() {
	certTree := BuildCertTree(".")
	if certTree == nil {
		ca := GenCACert(pkix.Name{CommonName: "TestCA",
			StreetAddress:      []string{"Calle Acme num. 1"},
			PostalCode:         []string{"12345"},
			Locality:           []string{"Acme City"},
			Province:           []string{"Acme County"},
			OrganizationalUnit: []string{"Acme Labs"},
			Organization:       []string{"Acme"},
			Country:            []string{"AcmeLand"}}, 4)
		GenCert("server.acme.com", ca, 2)
		certTree = BuildCertTree(".")
	}
	log.Print("CertTree:\n", certTree)
}

