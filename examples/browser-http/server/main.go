package main

import (
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gopherjs/gopherjs/js"
	"github.com/rtctunnel/rtctunnel/crypt"
	"github.com/rtctunnel/rtctunnel/peer"
)

var keypair crypt.KeyPair

func main() {
	saved := localStorageGet("keypair")
	if saved == "" {
		keypair = crypt.GenerateKeyPair()
		js.Global.Get("localStorage").Call("setItem", "keypair", keypair.Private.String()+"|"+keypair.Public.String())
	} else {
		private, err := crypt.NewKey(strings.Split(saved, "|")[0])
		if err != nil {
			panic(err)
		}
		public, err := crypt.NewKey(strings.Split(saved, "|")[1])
		if err != nil {
			panic(err)
		}
		keypair = crypt.KeyPair{Private: private, Public: public}
	}

	js.Global.Set("onload", onload)
}

func onload() {
	doc := js.Global.Get("document")
	body := doc.Call("getElementsByTagName", "body").Index(0)
	body.Get("style").Set("fontFamily", "monospace")

	p := doc.Call("createElement", "p")
	p.Set("innerHTML", "your public key: "+keypair.Public.String())
	body.Call("appendChild", p)

	form := doc.Call("createElement", "form")
	label := doc.Call("createElement", "label")
	label.Call("appendChild", doc.Call("createTextNode", "enter peer key:"))
	input := doc.Call("createElement", "input")
	input.Set("id", "peerPublicKey")
	input.Set("type", "text")
	input.Set("value", localStorageGet("peerpublickey"))
	label.Call("appendChild", doc.Call("createTextNode", " "))
	label.Call("appendChild", input)
	form.Call("appendChild", label)
	button := doc.Call("createElement", "input")
	button.Set("type", "submit")
	form.Call("appendChild", doc.Call("createTextNode", " "))
	form.Call("appendChild", button)
	body.Call("appendChild", form)

	form.Set("onsubmit", onsubmitpeerkey)

}

func onsubmitpeerkey(evt *js.Object) {
	evt.Call("preventDefault")

	value := js.Global.Get("document").Call("getElementById", "peerPublicKey").Get("value").String()
	peerPublicKey, err := crypt.NewKey(value)
	if err != nil {
		js.Global.Call("alert", err.Error())
		return
	}
	js.Global.Get("localStorage").Call("setItem", "peerpublickey", peerPublicKey.String())

	doc := js.Global.Get("document")
	body := doc.Call("getElementsByTagName", "body").Index(0)
	p := doc.Call("createElement", "p")
	p.Set("innerHTML", "run: rtctunnel add-route"+
		" --local-peer="+peerPublicKey.String()+
		" --local-port=8000"+
		" --remote-peer="+keypair.Public.String()+
		" --remote-port=80")
	body.Call("appendChild", p)

	go openConnection(peerPublicKey)
}

func localStorageGet(key string) string {
	res := js.Global.Get("localStorage").Call("getItem", key)
	if res.Bool() {
		return res.String()
	}
	return ""
}

func openConnection(peerPublicKey crypt.Key) {
	conn, err := peer.Open(keypair, peerPublicKey)
	if err != nil {
		js.Global.Call("alert", err.Error())
		return
	}
	defer conn.Close()

	dispatcher := peer.NewDispatcher(conn)
	li := dispatcher.Listen(80)
	runHTTP(li)
}

func runHTTP(li net.Listener) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello World")
	})
	err := http.Serve(li, mux)
	if err != nil {
		js.Global.Call("alert", err.Error())
		return
	}
}