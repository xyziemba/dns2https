package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/miekg/dns"
)

type dnsOverHTTPSResolver struct {
	Endpoint         string
	EdnsDisable      bool
	CheckingDisabled bool
}

func writeMsgOrCrash(w dns.ResponseWriter, msg *dns.Msg) {
	err := w.WriteMsg(msg)
	if err != nil {
		log.Fatalf("[FATAL ERROR] Failed to write output message.")
	}
}

func respondWithErrOrCrash(w dns.ResponseWriter, code int, req *dns.Msg) {
	res := &dns.Msg{}
	res.SetRcode(req, code)
	writeMsgOrCrash(w, res)
}

// Checks that this request is well formed and acceptable.
// Returns (bool, int)
func checkRequest(req *dns.Msg) (bool, int) {
	// DNS-Over-Http only supports one query at a time. Fail if there are multiple questions
	if len(req.Question) > 1 {
		log.Printf("[WARNING] DNS request with multiple questions:\n%+v\n", req)
		return false, dns.RcodeNotImplemented
	}

	if len(req.Question) == 0 {
		log.Printf("[WARNING] DNS request with no questions:\n%+v\n", req)
		return false, dns.RcodeFormatError
	}

	if !req.RecursionDesired {
		log.Printf("[WARNING] DNS request without recursion:\n%+v\n", req)
		return false, dns.RcodeNotImplemented // TODO: check rfc 1035 and see if "not implemented" is correct
	}

	if edns := req.IsEdns0(); (edns != nil) && edns.Do() {
		// this relay can't do DNSSEC because DNS-Over-HTTPS won't return RRSIGs
		log.Printf("[WARNING] DNS request requesting DNSSEC:\n%+v\n", req)
		return false, dns.RcodeFormatError
	}

	qClass := req.Question[0].Qclass
	if qClass != 1 {
		log.Printf("[WARNING] DNS request for class %d not supported", qClass)
		return false, dns.RcodeServerFailure
	}

	return true, dns.RcodeSuccess
}

func (resolver dnsOverHTTPSResolver) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {

	// Check that the request is well-formed and in our feature set
	if ok, rcode := checkRequest(req); !ok {
		respondWithErrOrCrash(w, rcode, req)
		return
	}

	// Create the endpoint
	q := req.Question[0]

	googleRequest := &googleDNSRequest{
		Name:             q.Name,
		Type:             q.Qtype,
		CheckingDisabled: resolver.CheckingDisabled,
	}
	if resolver.EdnsDisable {
		googleRequest.EdnsClientSubnet = "0.0.0.0/0"
	}

	url := resolver.Endpoint + googleRequest.ToQueryString()
	if verbose {
		log.Println("Requesting URL: " + url)
	}

	// HTTP get and JSON parse
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[ERROR] Request to '%s' failed with %s", url, err.Error())
		respondWithErrOrCrash(w, dns.RcodeServerFailure, req)
		return
	}
	defer resp.Body.Close()
	var out googleDNSResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if verbose {
		log.Printf("Upstream response: \n%+vs\n", out)
	}

	res, err := out.ReplyTo(req)
	if err != nil {
		log.Printf("[ERROR] Unable to create response.")
		respondWithErrOrCrash(w, dns.RcodeServerFailure, req)
		return
	}
	writeMsgOrCrash(w, res)
}

var verbose bool

func main() {

	port := flag.Uint("port", 53, "port to bind to")
	ednsDisable := flag.Bool("noedns", false, "disable EDNS")
	cd := flag.Bool("cd", false, "disable DNSSEC validation performed by upstream API")
	httpsEndpoint := flag.String("api", "https://dns.google.com/resolve", "resolver HTTPS address")
	flag.BoolVar(&verbose, "v", false, "print info on each request")
	launchd := flag.Bool("launchd", false, "use when starting with launchd to transfer ports")

	flag.Parse()

	resolver := &dnsOverHTTPSResolver{
		Endpoint:         *httpsEndpoint,
		EdnsDisable:      *ednsDisable,
		CheckingDisabled: *cd,
	}

	if *launchd {
		// get file descriptors from launchd
		udp, tcp, err := bootstrap()
		if err != nil {
			log.Fatalf("failed to start\n%s", err.Error())
		}

		udpServer := &dns.Server{
			PacketConn: udp,
			Handler:    resolver,
		}
		tcpServer := &dns.Server{
			Listener: tcp,
			Handler:  resolver,
		}

		if err := udpServer.ActivateAndServe(); err != nil {
			log.Fatalf("failed to set udp listener\n%s\n", err.Error())
		}
		if err := tcpServer.ActivateAndServe(); err != nil {
			log.Fatalf("failed to set tcp listener\n%s\n", err.Error())
		}
	} else {
		udpServer := &dns.Server{
			Addr:    fmt.Sprintf(":%d", *port),
			Net:     "udp",
			Handler: resolver,
		}
		tcpServer := &dns.Server{
			Addr:    fmt.Sprintf(":%d", *port),
			Net:     "tcp",
			Handler: resolver,
		}

		if err := udpServer.ListenAndServe(); err != nil {
			log.Fatalf("failed to set udp listener\n%s\n", err.Error())
		}
		if err := tcpServer.ListenAndServe(); err != nil {
			log.Fatalf("failed to set tcp listener\n%s\n", err.Error())
		}
	}
}
