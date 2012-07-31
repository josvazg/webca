// devsrv uses Russ's devwev for the development server
package main

import (
	"code.google.com/p/rsc/devweb/slave"
	"github.com/josvazg/webca"
	"net/http"
)

func main() {
	webca.PrepareServer(http.DefaultServeMux)
	webca.FakeLogin()
	slave.Main()
}
