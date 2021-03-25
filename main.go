package main

import (
	"flag"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/tidwall/gjson"
)

const internalIPv4 = "192.168.1.143"

var dnsClient = &dns.Client{
	Net:          "udp",
	Timeout:      5 * time.Second,
	ReadTimeout:  4 * time.Second,
	WriteTimeout: 1 * time.Second,
}

func isDomain(child, parrent string) bool {
	return child == parrent || (strings.HasSuffix(child, parrent) && child[len(child)-len(parrent)-1] == '.')
}

func getIP(q dns.Question) string {
	domain := strings.Trim(q.Name, ".")
	for ip, domains := range domainIPMap {
		for _, dm := range domains {
			neg := false
			if dm[0] == '!' {
				neg = true
				dm = dm[1:]
			}
			if dm[0] == '.' {
				if !isDomain(domain, dm[1:]) {
					continue
				}
			} else if domain != dm {
				continue
			}
			if neg {
				return "-"
			}
			// matched and not neg
			if ip[0] == 'A' {
				if ip[1] == 'A' {
					// IPv6
					if q.Qtype == dns.TypeAAAA {
						return ip[4:]
					}
					return ""
				}
				// IPv4
				if q.Qtype == dns.TypeA {
					return ip[1:]
				}
				return ""
			}
			// IPv4
			if q.Qtype == dns.TypeA {
				return ip
			}
			return ""
		}
	}

	// fallback Type AAAA, TXT,...
	return "-"
}

func fallbackQuery(msg *dns.Msg) ([]dns.RR, error) {
	r, _, err := dnsClient.Exchange(msg, "1.1.1.1:53")
	if err != nil {
		return nil, err
	}
	return r.Answer, nil
}

func responseQuery(reqMsg, replyMsg *dns.Msg) {
	fallbackMsg := reqMsg.Copy()
	fallbackMsg.Question = nil
	for _, q := range replyMsg.Question {
		ip := getIP(q)
		if len(ip) == 0 {
			// reject
			continue
		}
		if ip == "-" {
			// fallback
			fallbackMsg.Question = nil
			fallbackMsg.Question = append(fallbackMsg.Question, q)
			rr, err := fallbackQuery(fallbackMsg)
			if err != nil {
				log.Println("Query", dns.Type(q.Qtype).String(), ":", q.Name, "- fallback:", err)
				continue
			}
			if len(rr) == 0 {
				log.Println("Query", dns.Type(q.Qtype).String(), ":", q.Name, "- fallback nil")
			} else {
				replyMsg.Answer = append(replyMsg.Answer, rr...)
				rri := rr[len(rr)-1]
				log.Println("Query", dns.Type(q.Qtype).String(), ":", q.Name, "==", dns.Type(rri.Header().Rrtype).String(), "=>", rri.String()[len(rri.Header().String()):])
			}
			continue
		}
		log.Println("Query", dns.Type(q.Qtype).String(), ":", q.Name, "==>", ip)
		rr, err := dns.NewRR(q.Name + " A " + ip)
		if err != nil {
			log.Println("Failed to create RR:", q.Name, err)
		}
		replyMsg.Answer = append(replyMsg.Answer, rr)
	}
}

func handler(writer dns.ResponseWriter, reqMsg *dns.Msg) {
	replyMsg := &dns.Msg{}
	replyMsg.SetReply(reqMsg)
	replyMsg.Compress = false

	switch reqMsg.Opcode {
	case dns.OpcodeQuery:
		responseQuery(reqMsg, replyMsg)
		break
	default:
		rr, err := fallbackQuery(reqMsg)
		if err == nil {
			replyMsg.Answer = rr
		} else {
			log.Println("Fallback:", err)
		}
	}
	writer.WriteMsg(replyMsg)
}

var bindAddr = flag.String("addr", ":53", "UDP bind address")
var configFile = flag.String("config", "config.json", "Config JSON file")
var domainIPMap = map[string][]string{}

func main() {
	flag.Parse()

	content, err := os.ReadFile(*configFile)
	if err != nil {
		log.Panicln(err)
	}
	d := gjson.ParseBytes(content)
	if !d.IsObject() {
		log.Panicln("File errorr:", *configFile)
	}
	for key, val := range d.Map() {
		if !val.IsArray() {
			continue
		}
		arr, _ := domainIPMap[key]
		for _, d := range val.Array() {
			domain := d.String()
			if len(domain) < 2 {
				continue
			}
			arr = append(arr, domain)
		}
		sort.Slice(arr, func(i, j int) bool {
			if arr[i][0] == '!' {
				if arr[j][0] == '!' {
					return false
				}
				return true
			}
			return false
		})
		domainIPMap[key] = arr
	}

	log.Println(domainIPMap)

	dns.HandleFunc(".", handler)
	server := &dns.Server{
		Addr: *bindAddr,
		Net:  "udp",
	}
	log.Println("Started at:", *bindAddr)

	err = server.ListenAndServe()
	if err != nil {
		log.Println("Failed to start server:", err.Error())
	}
}
