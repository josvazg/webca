package webca

import (
	"net/http"
	"testing"
)

func dieOnError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func equal(s1, s2 session) bool {
	if len(s1)!=len(s2) {
		return false
	}
	for k,_:=range s1 {
		if s1[k]!=s2[k] {
			return false
		}
	}
	return true
}

func TestSessions(t *testing.T) {
	r,err := http.NewRequest("get", "/", nil)
	dieOnError(t, err)
	s, err := SessionFor(r)
	dieOnError(t, err)
	s["a"] = "A"
	s.Save()
	s2, err := SessionFor(r)
	dieOnError(t, err)
	if !equal(s,s2) {
		t.Fatal("Session save failed! s=%v vs s2=%v\n", s, s2)
	}
}

