package main

import (
	"flag"
	"log"

	"github.com/valyala/fasthttp"
	fastprefork "github.com/valyala/fasthttp/prefork"
)

var (
	fasthttpListen  string
	fasthttpPrefork bool
)

func init() {
	flag.StringVar(&fasthttpListen, `l`, `:8080`, `Listen address`)
	flag.BoolVar(&fasthttpPrefork, `prefork`, false, `Use prefork http`)

	flag.Parse()
}

func main() {
	isNotPreforkOrIsChild := !fasthttpPrefork || fastprefork.IsChild()
	handler := func(ctx *fasthttp.RequestCtx) {}

	if isNotPreforkOrIsChild {
		// init database

		// init handler
		handler = requestHandler
	}

	// init fasthttp server
	server := &fasthttp.Server{
		Name:    ``,
		Handler: handler,
	}

	listenAndServe := server.ListenAndServe
	if fasthttpPrefork {
		listenAndServe = fastprefork.New(server).ListenAndServe
	}

	log.Println(`Listen`, fasthttpListen)
	err := listenAndServe(fasthttpListen)
	if err != nil {
		log.Panicln(err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetBodyString(`Hello world`)
}
