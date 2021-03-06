package handler

import (
	"github.com/miekg/dns"
	"github.com/pritunl/pritunl-dns/constants"
	"github.com/pritunl/pritunl-dns/database"
	"github.com/pritunl/pritunl-dns/networks"
	"github.com/pritunl/pritunl-dns/question"
	"github.com/pritunl/pritunl-dns/resolver"
	"net"
	"time"
)

type Handler struct {
	reslvr *resolver.Resolver
}

func (h *Handler) handle(proto string, w dns.ResponseWriter, r *dns.Msg) {
	ques := question.NewQuestion(r.Question[0])

	subnet := ""
	if ip, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		subnet = networks.Find(ip.IP)
	}
	if ip, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		subnet = networks.Find(ip.IP)
	}

	if ques.IsIpQuery && ques.TopDomain == "vpn" {
		msg, err := h.reslvr.LookupUser(proto, ques, subnet, r)
		if err != nil {
			dns.HandleFailed(w, r)
			return
		}
		w.WriteMsg(msg)
	} else if ques.Qclass == dns.ClassINET && ques.Qtype == dns.TypePTR {
		msg, err := h.reslvr.LookupReverse(ques, r)
		if err != nil {
			dns.HandleFailed(w, r)
			return
		}
		w.WriteMsg(msg)
	} else {
		if subnet == "" {
			for subnet, _ = range database.DnsServers {
				break
			}
		}

		servers := database.DnsServers[subnet]
		res, err := h.reslvr.Lookup(proto, servers, r)
		if err != nil {
			dns.HandleFailed(w, r)
			return
		}

		w.WriteMsg(res)
	}
}

func (h *Handler) HandleTcp(w dns.ResponseWriter, r *dns.Msg) {
	h.handle("tcp", w, r)
}

func (h *Handler) HandleUdp(w dns.ResponseWriter, r *dns.Msg) {
	h.handle("udp", w, r)
}

func NewHandler(timeout, interval time.Duration) (h *Handler) {
	h = &Handler{
		reslvr: &resolver.Resolver{
			Timeout:        timeout,
			Interval:       interval,
			DefaultServers: constants.DefaultDnsServers,
		},
	}

	return
}
