package dnsproxylib

import (
	"fmt"
	"strconv"

	"github.com/miekg/dns"
)

type googleDNSRequest struct {
	Name             string `json:"name"`
	Type             uint16 `json:"type"`
	CheckingDisabled bool   `json:"cd"`
	EdnsClientSubnet string `json:"edns_client_subnet,omitempty"`
	// RandomPadding    string `json:"random_padding"`
}

func (g googleDNSRequest) ToQueryString() string {
	qs := "?name=" + g.Name + "&type=" + strconv.FormatUint(uint64(g.Type), 10)
	if g.CheckingDisabled {
		qs = qs + "&cd=true"
	}
	if g.EdnsClientSubnet != "" {
		qs = qs + "&edns_client_subnet=" + g.EdnsClientSubnet
	}
	return qs
}

type googleDNSResponse struct {
	Status     uint16
	TC         bool
	RD         bool
	RA         bool
	AD         bool
	CD         bool
	Question   googleDNSQuestions
	Answer     googleDNSAnswers
	Additional []interface{}
}

func (g googleDNSResponse) ReplyTo(origMsg *dns.Msg) (*dns.Msg, error) {
	outputMsg := &dns.Msg{}
	outputMsg.SetReply(origMsg)
	outputMsg.Rcode = (int)(g.Status)
	outputMsg.RecursionAvailable = true // we can only do recursive queries
	answers, err := g.Answer.ToRRs()
	if err != nil {
		return nil, err
	}
	outputMsg.Answer = answers
	return outputMsg, nil
}

type googleDNSQuestions []googleDNSQuestion

type googleDNSQuestion struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
}

type googleDNSAnswers []googleDNSAnswer

type googleDNSAnswer struct {
	Name string `json:"name"`
	Type uint16 `json:"type"`
	TTL  uint32 `json:"TTL"`
	Data string `json:"data"`
}

func (g googleDNSAnswer) ToRR() (dns.RR, error) {
	typeString := dns.TypeToString[g.Type]
	s := fmt.Sprintf("%s %d IN %s %s", g.Name, g.TTL, typeString, g.Data)
	return dns.NewRR(s)
}

func (gs googleDNSAnswers) ToRRs() ([]dns.RR, error) {
	answers := make([]dns.RR, len(gs), len(gs))
	for idx, g := range gs {
		rr, err := g.ToRR()
		if err != nil {
			return nil, err
		}
		answers[idx] = rr
	}
	return answers, nil
}
