package dnscache

import (
	"strings"

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
			name := strings.ToLower(r.Question[i].Name)
			entries, err := handler.cache.getCacheEntries(name)
			if err != nil {
				handler.cache.Logger.Error("Get cache entries failed", zap.String("error", err.Error()))
				msg.Answer = []dns.RR{}
				msg.Rcode = dns.RcodeServerFailure
				break
			}
			handler.cache.addCacheEntries(name, entries, &msg.Answer)
		}
		if len(msg.Answer) <= 0 {
			msg.Rcode = dns.RcodeServerFailure
		}
	}
	handler.cache.Logger.Debug("Answering Query", zap.Int("code", msg.Rcode), zap.Any("entries", msg.Answer))
	msg.SetReply(r)
	w.WriteMsg(&msg)
}
