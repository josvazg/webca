package main

import (
	"fmt"
	"net/smtp"
	"strings"
)

const (
	MAIL_LABEL = "WebCA"
)

type Mailer struct {
	Server, User, Passwd string
	bestAuth             smtp.Auth
}

func (m *Mailer) SendMail(to, subject, body string) error {
	host := m.Server
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	//log.Println("host=",host)
	auths := []smtp.Auth{m.bestAuth}
	msg := "from: \"" + MAIL_LABEL + "\" <" + m.User + ">\nto: " + to +
		"\nsubject: (" + MAIL_LABEL + ") " + subject + "\n\n" + body
	if m.bestAuth == nil {
		auths = []smtp.Auth{smtp.CRAMMD5Auth(m.User, m.Passwd),
			smtp.PlainAuth("", m.User, m.Passwd, host)}
	}
	var errs error
	for _, auth := range auths {
		err := smtp.SendMail(m.Server, auth, m.User, []string{to}, ([]byte)(msg))
		if err == nil {
			m.bestAuth = auth
			return nil
		} else {
			errs = fmt.Errorf("%v%v\n", errs, err)
		}
	}
	return errs
}

/*
func read(msg string, ptr interface{}) {
	fmt.Print(msg)
	os.Stdout.Sync()
	fmt.Scan(ptr)
}

func ask() Mailer {
	mailer := Mailer{}
	read("Mail Server: ", &mailer.Server)
	read(" Email user: ", &mailer.User)
	read("   Password: ", &mailer.Passwd)
	f, err := os.OpenFile(MAILCFG_FILE, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	handleFatal(err)
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(mailer)
	handleFatal(err)
	return mailer
}

func setup() Mailer {
	_, err := os.Stat(MAILCFG_FILE)
	if os.IsNotExist(err) {
		return ask()
	}
	handleFatal(err)

	f, err := os.Open(MAILCFG_FILE)
	handleFatal(err)
	defer f.Close()
	dec := gob.NewDecoder(f)
	if dec == nil {
		log.Fatalf("(Warning) Could not decode " + MAILCFG_FILE + "!")
	}
	mailer := Mailer{}
	err = dec.Decode(&mailer)
	handleFatal(err)
	return mailer
}

func main() {
	subject:="Test notification"
	to:="josvazg+webca@gmail.com"
	err:=mailer.sendMail(to, subject, "Notification!")
	if err!=nil {
		log.Fatal(err)
	} else {
		log.Print("Mail '"+subject+"' Sent to "+to)
	}
}
*/

