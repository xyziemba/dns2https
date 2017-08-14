package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/miekg/dns"
	"github.com/xyziemba/dnsproxy/dnsproxylib"
)

func main() {

	port := flag.Uint("port", 53, "port to bind to")
	ednsDisable := flag.Bool("noedns", false, "disable EDNS")
	cd := flag.Bool("cd", false, "disable DNSSEC validation performed by upstream API")
	httpsEndpoint := flag.String("api", "https://dns.google.com/resolve", "resolver HTTPS address")
	verbose := flag.Bool("v", false, "print info on each request")
	launchd := flag.Bool("launchd", false, "use when starting with launchd to transfer ports")

	flag.Parse()

	resolver := &dnsproxylib.DnsOverHTTPSResolver{
		Endpoint:         *httpsEndpoint,
		EdnsDisable:      *ednsDisable,
		CheckingDisabled: *cd,
		Debug:            *verbose,
	}

	var udpServer, tcpServer *dns.Server

	if *launchd {
		// get file descriptors from launchd
		udp, tcp, err := bootstrap()
		if err != nil {
			log.Fatalf("failed to start\n%s", err.Error())
		}

		udpServer = &dns.Server{
			PacketConn: udp,
			Handler:    resolver,
		}
		tcpServer = &dns.Server{
			Listener: tcp,
			Handler:  resolver,
		}

		go func() {
			if err := udpServer.ActivateAndServe(); err != nil {
				log.Fatalf("failed to set udp listener\n%s\n", err.Error())
			}
		}()
		go func() {
			if err := tcpServer.ActivateAndServe(); err != nil {
				log.Fatalf("failed to set tcp listener\n%s\n", err.Error())
			}
		}()
	} else {
		udpServer = &dns.Server{
			Addr:    fmt.Sprintf(":%d", *port),
			Net:     "udp",
			Handler: resolver,
		}
		tcpServer = &dns.Server{
			Addr:    fmt.Sprintf(":%d", *port),
			Net:     "tcp",
			Handler: resolver,
		}

		go func() {
			if err := udpServer.ListenAndServe(); err != nil {
				log.Fatalf("failed to set udp listener\n%s\n", err.Error())
			}
		}()
		go func() {
			if err := tcpServer.ListenAndServe(); err != nil {
				log.Fatalf("failed to set tcp listener\n%s\n", err.Error())
			}
		}()
	}

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	sig := <-shutdown
	log.Printf("Shutting down. Received signal: %s", sig.String())

	err := udpServer.Shutdown()
	if err != nil {
		log.Fatalf("udp server shutdown failed. Exiting ungracefully")
	}
	err = tcpServer.Shutdown()
	if err != nil {
		log.Fatalf("tcp server shutdown failed. Exiting ungracefully")
	}
}
