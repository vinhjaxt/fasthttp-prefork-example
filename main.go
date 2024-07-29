package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"

	"github.com/valyala/fasthttp"
)

var listen = flag.String(`l`, `:80`, `HTTP Listen address`)
var prefork = flag.Bool(`prefork`, false, `Prefork`)

func main() {
	flag.Parse()

	// Server
	var err error
	var ln net.Listener
	var fd *os.File

	if !*prefork || !PreforkIsChild() {
		log.Println(`[master] init listener`)
		// master or not prefork
		if strings.HasPrefix(*listen, `unix:`) {
			unixFile := (*listen)[5:]
			os.Remove(unixFile)
			ln, err = net.Listen(`unix`, unixFile)
			os.Chmod(unixFile, os.ModePerm)
			if ln != nil {
				log.Println(`[master] listening:`, unixFile)
				fln := ln.(*net.UnixListener)
				fd, err = fln.File()
			}
		} else {
			ln, err = net.Listen(`tcp`, *listen)
			if ln != nil {
				log.Println(`[master] listening:`, ln.Addr().String())
				fln := ln.(*net.TCPListener)
				fd, err = fln.File()
			}
		}
		if err != nil {
			log.Panicln(err)
		}
		if ln == nil {
			log.Panicln(`[master] error listening:`, *listen)
		}
	}

	srv := &fasthttp.Server{
		Name:               `nginx`,
		Handler:            requestHandler,
		MaxRequestBodySize: 200 * 1024 * 1024, // 200 MB
	}

	if !*prefork {
		log.Println(`[master] single process serving`)
		log.Panicln(srv.Serve(ln))
	}

	if PreforkIsChild() {
		// child
		ln, err = PreforkGetListenerFd()
		if err != nil {
			log.Panicln(`[child] listener:`, err)
		}
		log.Println(`[child] running`)
		log.Panicln(srv.Serve(ln))
	} else {
		// master
		if fd == nil {
			log.Panicln(`[master] listen error, fd = nil`)
		}
		log.Println(`[master] prefork`)
		log.Panicln(Prefork([]*os.File{fd}, 0))
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	ctx.Response.SetBodyString(`Hello world`)
}
