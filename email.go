package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/smtp"
	"os"
)

const (
	MAILCFG_FILE = "mail.cfg"
)

type Mailer struct {
	Server, User, Passwd string
}

func (m *Mailer) sendMail(to, subject, msg string) {
	auth := smtp.PlainAuth("", m.User, m.Passwd, m.Server)
	err := smtp.SendMail(m.Server, auth, m.User, []string{to}, ([]byte)(msg))
	if err != nil {
		log.Fatalf("Could not send email: ", err)
	}
}

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
	mailer := setup()
	mailer.sendMail("josvazg+webca@gmail.com", "Test notification", "Notification!")
}

