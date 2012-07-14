package webca

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"log"
	"sync"
)

const (
	SESSIONID="_SESSION_ID"
)

// session type
type session map[string]interface{}

// sessions holds all sessions
var sessions map[string]session

// mutex lock for session access
var smutex sync.RWMutex

// SessionFor gets a session bound to a Request by and Session ID
func SessionFor(r *http.Request) (session,error) {
	id:=sessionId(r)
	smutex.RLock()
	defer smutex.RUnlock()
	if id=="" {
		nid,err:=genSessionId()
		if err!=nil {
			return nil,err
		}
		id=nid
	}
	if sessions==nil {
		sessions=make(map[string]session)
	}
	s:=sessions[id]
	log.Println("* s=",s)
	if s==nil || len(s)==0 {
		s=session{}
		s[SESSIONID]=id
		sessions[id]=s
	}
	log.Println("session=",s)
	return clone(s),nil // this copy allows concurrent session access
}

// Save stores the session state 
func (s session) Save() {
	smutex.Lock()
	defer smutex.Unlock()
	sessions[SESSIONID]=clone(s)
	log.Println("NEW sessions=",sessions)
}

// clone makes a copy of a session and returns it
func clone(s session) session {
	c:=session{}
	for k,v:=range s {
		c[k]=v
	}
	return c
}

// sessionId gets a session id from the request
func sessionId(r *http.Request) string {
	return r.FormValue(SESSIONID)
}

// genSessionId generates a new session ID
func genSessionId() (string,error) {
	uuid := make([]byte, 16)
 	n, err := rand.Read(uuid)
 	if n != len(uuid) || err != nil {
 		return "", err
 	}
 	// TODO: verify the two lines implement RFC 4122 correctly
 	uuid[8] = 0x80 // variant bits see page 5
 	uuid[4] = 0x40 // version 4 Pseudo Random, see page 7
 	return hex.EncodeToString(uuid), nil
}
