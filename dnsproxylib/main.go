package dnsproxylib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/miekg/dns"
)

type DnsOverHTTPSResolver struct {
	Endpoint         string
	EdnsDisable      bool
	CheckingDisabled bool
	BootstrapDNSIPs  []string
	Debug            bool

	resolver   *SingleARecordResolver
	httpClient *http.Client
}

func (r *DnsOverHTTPSResolver) BootstrapResolver() (*SingleARecordResolver, error) {
	if r.resolver == nil {
		endpointURL, err := url.Parse(r.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("invalid httpsEndpoint '%s'", endpointURL)
		}
		endpointHost := endpointURL.Host
		r.resolver = NewSingleARecordResolver(endpointHost)
	}
	return r.resolver, nil
}

func (r *DnsOverHTTPSResolver) HTTPClient() *http.Client {
	if r.httpClient == nil {
		r.httpClient = &http.Client{
			Transport: &http.Transport{
				DialContext: dialContext,
			},
		}
	}
	return r.httpClient
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

func (resolver DnsOverHTTPSResolver) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {

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
	if req.CheckingDisabled {
		googleRequest.CheckingDisabled = req.CheckingDisabled
	}

	url := resolver.Endpoint + googleRequest.ToQueryString()
	if resolver.Debug {
		log.Println("Requesting URL: " + url)
	}

	// HTTP get and JSON parse
	httpReq, err := http.NewRequest("GET", url, http.NoBody)
	resp, err := resolver.HTTPClient().Do(httpReq)

	if err != nil {
		log.Printf("[ERROR] Request to '%s' failed with %s", url, err.Error())
		respondWithErrOrCrash(w, dns.RcodeServerFailure, req)
		return
	}
	defer resp.Body.Close()
	var out googleDNSResponse
	json.NewDecoder(resp.Body).Decode(&out)
	if resolver.Debug {
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

var ipResolvingTransport http.RoundTripper = &http.Transport{
	// Custom DialContext allows us to use a custom resolver
	//
	// TODO: using a new Dialer with a custom resolver may be more robust
	DialContext: dialContext,
}

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ipHost, err := NewSingleARecordResolver(host).Resolve()
	if err != nil {
		return nil, err
	}

	ipAddr := net.JoinHostPort(ipHost, port)
	return http.DefaultTransport.(*http.Transport).DialContext(ctx, network, ipAddr)
}
