package acme

import (
 "net"
 "sync/atomic"
 "testing"
 "github.com/miekg/dns"
)

func TestTruncatedUDPResponseRetriesOverTCP(t *testing.T) {
 tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 udpListener, err := net.ListenPacket("udp", tcpListener.Addr().String())
 if err != nil { tcpListener.Close(); t.Fatal(err) }
 var tcpQueries atomic.Int32
 udp := &dns.Server{PacketConn: udpListener, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, q *dns.Msg) {
  response := new(dns.Msg); response.SetReply(q); response.Truncated = true
  if err := w.WriteMsg(response); err != nil { t.Error(err) }
 })}
 tcp := &dns.Server{Listener: tcpListener, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, q *dns.Msg) {
  tcpQueries.Add(1)
  response := new(dns.Msg); response.SetReply(q)
  response.Answer = []dns.RR{&dns.TXT{Hdr: dns.RR_Header{Name:q.Question[0].Name, Rrtype:dns.TypeTXT, Class:dns.ClassINET}, Txt:[]string{"challenge-value"}}}
  if err := w.WriteMsg(response); err != nil { t.Error(err) }
 })}
 udpReady, tcpReady := make(chan struct{}), make(chan struct{})
 udp.NotifyStartedFunc = func(){close(udpReady)}
 tcp.NotifyStartedFunc = func(){close(tcpReady)}
 go udp.ActivateAndServe(); go tcp.ActivateAndServe()
 <-udpReady; <-tcpReady
 defer udp.Shutdown(); defer tcp.Shutdown()
 response, err := dnsQuery("example.test.", dns.TypeTXT, []string{tcpListener.Addr().String()}, true)
 if err != nil { t.Fatal(err) }
 if tcpQueries.Load()!=1 || response==nil || len(response.Answer)!=1 { t.Fatalf("expected complete TCP answer after UDP truncation: %#v",response) }
 answer, ok := response.Answer[0].(*dns.TXT)
 if !ok || len(answer.Txt)!=1 || answer.Txt[0]!="challenge-value" { t.Fatalf("unexpected answer: %#v",response.Answer) }
}
