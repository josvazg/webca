package webca

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"os"
	"testing"
)

func NewCert(name string, childs ... *Cert) *Cert {
	return &Cert{Crt: &x509.Certificate{
					Subject: pkix.Name{
						CommonName: name}}, 
				Childs: childs}
}

func loadTestData() *Certree {
	ct := NewCertree()
	ct.roots=[]*Cert{
		NewCert("TestCA1",
			NewCert("Intermediate1",
				NewCert("server1"),
				NewCert("server2"),
				NewCert("server3"))),
		NewCert("TestCA2",
			NewCert("Intermediate2",
				NewCert("2serverA"),
				NewCert("2serverB"))),
	}
	ct.foreign=[]*Cert{
		NewCert("SomeCA0",
			NewCert("externalserver1")),
		NewCert("SomeCA1",
			NewCert("externalserverA")),
	}
	return ct
}

func gen(t *testing.T, cert *Cert) *Cert {
	var gcert *Cert
	var err error
	if cert.Parent == nil {
		gcert, err = GenCACert(cert.Crt.Subject, 1095)
		dieOnError(t, err)
	} else {
		gcert, err = GenCert(cert.Parent, cert.Crt.Subject.CommonName, 1095)
		dieOnError(t, err)
	}
	for i, crt := range cert.Childs {
		crt.Parent = gcert
		cert.Childs[i] = gen(t, crt)
	}
	cert.Crt = gcert.Crt
	cert.Key = gcert.Key
	return cert
}

func TestCA(t *testing.T) {
	ct0 := loadTestData()
	dieOnError(t, os.MkdirAll("tests", 0750))
	dieOnError(t, os.Chdir("tests"))
	for i, crt := range ct0.roots {
		ct0.roots[i] = gen(t, crt)
	}
	for _, crt := range ct0.foreign {
		gen(t, crt)
	}
	dieOnError(t, os.Remove("SomeCA0.key.pem"))
	dieOnError(t, os.Remove("SomeCA1.key.pem"))
	ct:=LoadCertree(".")
	s0 := ct0.String()
	s := ct.String()
	if s != s0 {
		t.Log(ct0, "\nVs ")
		t.Log(ct)
		t.Fatal("Template test and reloaded tree do not match!")
	}
	dieOnError(t, os.Chdir(".."))
	dieOnError(t, os.RemoveAll("tests"))

	//log.Print(certTree)
	//log.Print("CertTree.first:\n", certTree.first)
	//RenewCert(nil, certTree.first.ca)
	//RenewCert(certTree.first.ca, certTree.first.Certs[0])
	//certTree = LoadCertTree(".")
	//log.Print("Renewed CertTree:\n", certTree)
}

