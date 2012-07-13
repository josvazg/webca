package webca

import (
	"net/http"
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
func SessionFor(r *http.Request) session {
	id:=sessionId(r)
	smutex.RLock()
	defer smutex.RUnlock()
	if sessions==nil {
		sessions=make(map[string]session)
	}
	s:=sessions[id]
	if s==nil {
		s:=session{}
		s[SESSIONID]=id
		sessions[id]=s
	}
	return clone(s) // this copy allows for concurrent session access
}

// Save stores the session state 
func (s session) Save() {
	smutex.Lock()
	defer smutex.Unlock()
	sessions[SESSIONID]=clone(s)
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

