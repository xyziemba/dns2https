package main

import (
	"errors"
	"time"

	"github.com/miekg/dns"
)

// SingleARecordResolver resolves a single 'A' record using a DNS
// server with a known IP address. It caches the returned value
// until the TTL has expired.
type SingleARecordResolver struct {
	Nameservers []string // IP addresses of nameservers to query. Cannot be nil or empty. Must include port numbers
	Client      dns.Client

	recordName string
	timeout    time.Time
	cachedIP   string
}

// NewSingleARecordResolver creates a resolver that will look up
// recordName using Google DNS.
func NewSingleARecordResolver(recordName string) *SingleARecordResolver {
	return &SingleARecordResolver{
		recordName:  recordName,
		Nameservers: []string{"8.8.8.8:53", "8.8.4.4:53"}, // sensible default for Google DNS
	}
}

// Resolve returns the IPv4 address for the recordName used when
// creating this struct.
//
// If Resolve has been called before, then a cached value may be returned.
func (s *SingleARecordResolver) Resolve() (string, error) {
	if s.cachedIP != "" && s.timeout.After(time.Now()) {
		return s.cachedIP, nil
	}

	msg := new(dns.Msg)
	msg = msg.SetQuestion(s.recordName+".", (uint16)(dns.TypeA))

	for _, ipAddr := range s.Nameservers {
		response, _, err := s.Client.Exchange(msg, ipAddr)
		if err != nil || len(response.Answer) == 0 {
			continue
		}

		record := response.Answer[0].(*dns.A)
		ip := record.A.String()
		s.cachedIP = ip
		s.timeout = time.Now().Add((time.Duration)(record.Hdr.Ttl))
		return ip, nil
	}

	return "", errors.New("Unable to resolve")
}
