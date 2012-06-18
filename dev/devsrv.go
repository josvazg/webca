// devsrv uses Russ's devwev for the development server
package main

import (
	"code.google.com/p/rsc/devweb/slave"
	"github.com/josvazg/webca"
)

func main() {
	webca.PrepareServer()
	slave.Main()
}

