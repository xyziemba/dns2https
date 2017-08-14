package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/miekg/dns"
	"github.com/xyziemba/dns2https/dns2httpslib"
)

func main() {

	port := flag.Uint("port", 53, "port to bind to")
	ednsDisable := flag.Bool("noedns", false, "disable EDNS")
	cd := flag.Bool("cd", false, "disable DNSSEC validation performed by upstream API")
	httpsEndpoint := flag.String("api", "https://dns.google.com/resolve", "resolver HTTPS address")
	verbose := flag.Bool("v", false, "print info on each request")
	launchd := flag.Bool("launchd", false, "use when starting with launchd to transfer ports")
	selftest := flag.Bool("selftest", false, "run a simple self test on port 8053")

	flag.Parse()

	resolver := &dns2httpslib.DnsOverHTTPSResolver{
		Endpoint:         *httpsEndpoint,
		EdnsDisable:      *ednsDisable,
		CheckingDisabled: *cd,
		Debug:            *verbose,
	}

	if *launchd && *selftest {
		log.Fatal("-selftest and -launchd are mutually exclusive")
	}

	if *selftest {
		*port = 8053
	}

	var udpServer, tcpServer *dns.Server
	started := make(chan string, 1) // buffered because this may not be read

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
			Addr:              fmt.Sprintf(":%d", *port),
			Net:               "udp",
			Handler:           resolver,
			NotifyStartedFunc: func() { started <- "started" },
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

	if !*selftest {
		shutdown := make(chan os.Signal)
		signal.Notify(shutdown, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
		sig := <-shutdown
		log.Printf("Shutting down. Received signal: %s", sig.String())
	} else {
		<-started // wait to start udp server

		client := &dns.Client{}
		msg := &dns.Msg{}
		msg.SetQuestion("brew.sh.", dns.TypeA)
		msg.RecursionDesired = true

		_, _, err := client.Exchange(msg, fmt.Sprintf(":%d", *port))
		if err != nil {
			log.Fatalf("selftest failed\n%s\n", err.Error())
		}
	}

	err := udpServer.Shutdown()
	if err != nil {
		log.Fatalf("udp server shutdown failed. Exiting ungracefully")
	}
	err = tcpServer.Shutdown()
	if err != nil {
		log.Fatalf("tcp server shutdown failed. Exiting ungracefully")
	}
}
