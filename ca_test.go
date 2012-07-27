package webca

import (
	"crypto/x509"
	"crypto/x509/pkix"
	//"os"
	"testing"
)

func buildPath(c *Cert, names ...string) *Cert {
	tmpl := pkix.Name{StreetAddress: []string{"Acme st. num. 23"},
		PostalCode:         []string{"12345"},
		Locality:           []string{"Acme City"},
		Province:           []string{"Acme County"},
		OrganizationalUnit: []string{"Acme Labs"},
		Organization:       []string{"Acme"},
		Country:            []string{"AcmeLand"}}
	if c == nil {
		c = &Cert{}
	}
	crt := c
	p := (*Cert)(nil)
	for _, name := range names {
		if crt == nil {
			crt = &Cert{}
			crt.Parent = crt
		}
		sbj := copyName(tmpl)
		sbj.CommonName = name
		crt.Crt = &x509.Certificate{Subject: sbj}
		crt.Parent = p
		if p != nil {
			p.Childs = append(p.Childs, crt)
		}
		p = crt
		crt = nil
	}
	return c
}

func loadTestData() *Certree {
	ct := NewCertree()
	root1 := buildPath(nil, "TestCA1", "Intermediate1", "server1")
	buildPath(root1, "TestCA1", "Intermediate1", "server2")
	buildPath(root1, "TestCA1", "Intermediate2", "serverA")
	buildPath(root1, "TestCA1", "Intermediate2", "serverB")
	ct.roots = append(ct.roots, root1)
	root2 := buildPath(nil, "TestCA2", "Intermediate2", "2server1")
	buildPath(root2, "TestCA2", "Intermediate1", "2server2")
	buildPath(root2, "TestCA2", "Intermediate2", "2serverA")
	ct.roots = append(ct.roots, root2)
	ct.foreign = append(ct.foreign, buildPath(nil, "SomeCA0", "externalserver1"))
	ct.foreign = append(ct.foreign, buildPath(nil, "SomeCA1", "externalserverA"))
	return ct
}

func gen(t *testing.T, cert *Cert) *Cert {
	t.Log("Generating " + cert.Crt.Subject.CommonName + "...")
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
	t.Log(ct0)
	t.Fatal("Fake failure")
	/*dieOnError(t, os.MkdirAll("tests", 0750))
	dieOnError(t, os.Chdir("tests"))
	for i, crt := range ct0.roots {
		ct0.roots[i] = gen(t, crt)
	}
	for _, crt := range ct0.foreign {
		gen(t, crt)
	}
	dieOnError(t, os.Remove("SomeCA0.pem"))
	dieOnError(t, os.Remove("SomeCA0.key.pem"))
	dieOnError(t, os.Remove("SomeCA1.pem"))
	dieOnError(t, os.Remove("SomeCA1.key.pem"))
	ct := LoadCertree(".")

	s0 := ct0.String()
	s := ct.String()
	if s != s0 {
		t.Log(ct0, "\nVs ")
		t.Log(ct)
		t.Fatal("Template test and reloaded tree do not match!")
	}
	dieOnError(t, os.Chdir(".."))
	dieOnError(t, os.RemoveAll("tests"))*/

	//log.Print(certTree)
	//log.Print("CertTree.first:\n", certTree.first)
	//RenewCert(nil, certTree.first.ca)
	//RenewCert(certTree.first.ca, certTree.first.Certs[0])
	//certTree = LoadCertTree(".")
	//log.Print("Renewed CertTree:\n", certTree)
}

