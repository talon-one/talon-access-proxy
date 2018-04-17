package dnscache

import (
	"github.com/miekg/dns"
	"go.uber.org/zap"
)

type handler struct {
	cache *DNSCache
}

func (handler handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	var msg dns.Msg
	if len(r.Question) <= 0 {
		msg.Rcode = dns.RcodeServerFailure
	} else {
		for i := 0; i < len(r.Question); i++ {
			entries, err := handler.cache.Lookup(r.Question[i].Name, r.Question[i].Qclass, r.Question[i].Qtype)
			if err != nil {
				handler.cache.Logger.Error("Get cache entries failed", zap.String("error", err.Error()))
				msg.Answer = []dns.RR{}
				msg.Rcode = dns.RcodeServerFailure
				break
			}
			handler.cache.addCacheEntries(r.Question[i].Name, entries, &msg.Answer)
		}
		if len(msg.Answer) <= 0 {
			msg.Rcode = dns.RcodeServerFailure
		}
	}
	handler.cache.Logger.Debug("Answering Query", zap.Int("code", msg.Rcode), zap.Any("entries", msg.Answer))
	msg.SetReply(r)
	w.WriteMsg(&msg)
}
