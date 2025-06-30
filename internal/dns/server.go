// internal/dns/server.go
package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"

	"errantdns.io/internal/models"
	"errantdns.io/internal/resolver"
	"errantdns.io/internal/storage"
	"errantdns.io/internal/logging"
)

// Server represents a DNS server instance
type Server struct {
	resolver  *resolver.Resolver
	udpServer *dns.Server
	tcpServer *dns.Server
	port      string

	// Server statistics
	stats Stats
}

// Stats holds DNS server statistics
type Stats struct {
	QueriesReceived int64
	QueriesAnswered int64
	QueriesNXDomain int64
	QueriesError    int64

	// Query type breakdown
	TypeA     int64
	TypeAAAA  int64
	TypeCNAME int64
	TypeMX    int64
	TypeTXT   int64
	TypeNS    int64
	TypeSRV   int64
	TypeSOA   int64
	TypePTR   int64
	TypeCAA   int64
	TypeOther int64
}

// Config holds configuration for the DNS server
type Config struct {
	Port          string
	UDPTimeout    time.Duration
	TCPTimeout    time.Duration
	MaxConcurrent int
}

// DefaultConfig returns DNS server config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Port:          "5353",
		UDPTimeout:    5 * time.Second,
		TCPTimeout:    10 * time.Second,
		MaxConcurrent: 1000,
	}
}

// NewServer creates a new DNS server instance
func NewServer(storage storage.Storage, config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	resolverConfig := &resolver.Config{}
	dnsResolver := resolver.NewResolver(storage, resolverConfig)

	server := &Server{
		resolver: dnsResolver,
		port:     config.Port,
	}

	// Set up DNS request handler
	dns.HandleFunc(".", server.handleDNSRequest)

	// Create UDP server
	server.udpServer = &dns.Server{
		Addr:         "0.0.0.0:" + config.Port,
		Net:          "udp4",
		ReadTimeout:  config.UDPTimeout,
		WriteTimeout: config.UDPTimeout,
	}

	// Create TCP server
	server.tcpServer = &dns.Server{
		Addr:         "0.0.0.0:" + config.Port,
		Net:          "tcp4",
		ReadTimeout:  config.TCPTimeout,
		WriteTimeout: config.TCPTimeout,
	}

	return server
}

// Start starts both UDP and TCP DNS servers
func (s *Server) Start(ctx context.Context) error {
	logging.Info("dns", "Starting DNS server on port %s", s.port)

	// Start UDP server in goroutine
	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			logging.Info("dns", "UDP server error: %v", "details", fmt.Sprintf("UDP server error: %v", err))
		}
	}()

	// Start TCP server in goroutine
	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			logging.Info("dns", "TCP server error: %v", "details", fmt.Sprintf("TCP server error: %v", err))
		}
	}()

	logging.Info("dns", "DNS server started successfully")

	// Wait for context cancellation
	<-ctx.Done()
	logging.Info("dns", "DNS server shutting down...")

	return s.Stop()
}

// Stop gracefully stops both DNS servers
func (s *Server) Stop() error {
	var udpErr, tcpErr error

	if s.udpServer != nil {
		udpErr = s.udpServer.Shutdown()
	}

	if s.tcpServer != nil {
		tcpErr = s.tcpServer.Shutdown()
	}

	// Return first error encountered
	if udpErr != nil {
		return fmt.Errorf("UDP server shutdown error: %w", udpErr)
	}
	if tcpErr != nil {
		return fmt.Errorf("TCP server shutdown error: %w", tcpErr)
	}

	logging.Info("dns", "DNS server stopped successfully")
	return nil
}

// GetStats returns current server statistics
func (s *Server) GetStats() Stats {
	return s.stats
}

// handleDNSRequest processes incoming DNS requests
func (s *Server) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	s.stats.QueriesReceived++

	// Create response message
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true
	msg.RecursionAvailable = false

	// Process each question in the request
	for _, question := range r.Question {
		if err := s.processQuestion(&msg, &question); err != nil {
			logging.Error("dns", "Error processing question %s %s: %v", nil,
				question.Name, dns.TypeToString[question.Qtype], err)
			msg.Rcode = dns.RcodeServerFailure
			s.stats.QueriesError++
		}
	}

	// Update statistics based on response code
	switch msg.Rcode {
	case dns.RcodeSuccess:
		if len(msg.Answer) > 0 {
			s.stats.QueriesAnswered++
		} else {
			s.stats.QueriesNXDomain++
		}
	case dns.RcodeNameError:
		s.stats.QueriesNXDomain++
	default:
		s.stats.QueriesError++
	}

	// Send the response
	if err := w.WriteMsg(&msg); err != nil {
		logging.Error("dns", "Failed to write DNS response: %v", nil, err)
		s.stats.QueriesError++
	}
}

// processQuestion handles a single DNS question
func (s *Server) processQuestion(msg *dns.Msg, question *dns.Question) error {
	// Extract query details
	queryName := question.Name
	queryType := dns.TypeToString[question.Qtype]

	logging.Debug("dns", "DNS Query received", "domain", queryName, "type", queryType)

	// Update type statistics
	s.updateTypeStats(question.Qtype)

	// Convert to our internal query format
	query := models.NewLookupQuery(queryName, queryType)

	// Look up the record in storage
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Handle record types that should return multiple records
	if question.Qtype == dns.TypeSRV || question.Qtype == dns.TypeMX || question.Qtype == dns.TypeNS {
		// For SRV, MX, and NS records, return all records
		records, err := s.resolver.ResolveAll(ctx, query)
		if err != nil {
			return fmt.Errorf("resolver lookup failed: %w", err)
		}

		if len(records) == 0 {
			logging.Info("dns", "No records found for %s %s", "details", fmt.Sprintf("No records found for %s %s", queryName, queryType))
			msg.Rcode = dns.RcodeNameError
			return nil
		}

		// Convert all records to DNS resource records
		for _, record := range records {
			rr, err := s.createResourceRecord(record, question.Qtype)
			if err != nil {
				return fmt.Errorf("failed to create resource record: %w", err)
			}

			if rr != nil {
				msg.Answer = append(msg.Answer, rr)
				logging.Info("dns", "Answered %s %s -> %s (priority: %d) [DB]", "details", fmt.Sprintf("Answered %s %s -> %s (priority: %d) [DB]", queryName, queryType, record.Target, record.Priority))
			}
		}

		return nil
	}

	record, err := s.resolver.Resolve(ctx, query)
	if err != nil {
		return fmt.Errorf("resolver lookup failed: %w", err)
	}

	// Handle no record found
	if record == nil {
		logging.LogNXDOMAIN(queryName, queryType, 0)
		msg.Rcode = dns.RcodeNameError
		return nil
	}

	// Convert to DNS resource record
	rr, err := s.createResourceRecord(record, question.Qtype)
	if err != nil {
		return fmt.Errorf("failed to create resource record: %w", err)
	}

	if rr != nil {
		msg.Answer = append(msg.Answer, rr)
		logging.Info("dns", "Answered %s %s -> %s [DB]", "details", fmt.Sprintf("Answered %s %s -> %s [DB]", queryName, queryType, record.Target))
	} else {
		// Record type mismatch
		log.Printf("Record type mismatch for %s: found %s, requested %s",
			queryName, record.RecordType, queryType)
		msg.Rcode = dns.RcodeNameError
	}

	return nil
}

// createResourceRecord converts our internal record to a DNS resource record
func (s *Server) createResourceRecord(record *models.DNSRecord, qtype uint16) (dns.RR, error) {
	recordType := models.RecordType(record.RecordType)

	switch recordType {
	case models.RecordTypeA:
		if qtype == dns.TypeA {
			ip := net.ParseIP(record.Target)
			if ip == nil || ip.To4() == nil {
				return nil, fmt.Errorf("invalid IPv4 address: %s", record.Target)
			}
			return &dns.A{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				A: ip.To4(),
			}, nil
		}

	case models.RecordTypeAAAA:
		if qtype == dns.TypeAAAA {
			ip := net.ParseIP(record.Target)
			if ip == nil || ip.To4() != nil {
				return nil, fmt.Errorf("invalid IPv6 address: %s", record.Target)
			}
			return &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				AAAA: ip.To16(),
			}, nil
		}

	case models.RecordTypeCNAME:
		if qtype == dns.TypeCNAME {
			return &dns.CNAME{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeCNAME,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				Target: dns.Fqdn(record.Target),
			}, nil
		}

	case models.RecordTypeTXT:
		if qtype == dns.TypeTXT {
			return &dns.TXT{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				Txt: []string{record.Target},
			}, nil
		}

	case models.RecordTypeMX:
		if qtype == dns.TypeMX {
			return &dns.MX{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeMX,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				Mx:         dns.Fqdn(record.Target),
				Preference: uint16(record.Priority),
			}, nil
		}

	case models.RecordTypeNS:
		if qtype == dns.TypeNS {
			return &dns.NS{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeNS,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				Ns: dns.Fqdn(record.Target),
			}, nil
		}

	case models.RecordTypeSOA:
		if qtype == dns.TypeSOA {
			return &dns.SOA{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeSOA,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				Ns:      dns.Fqdn(record.Target),
				Mbox:    dns.Fqdn(record.Mbox),
				Serial:  record.Serial,
				Refresh: record.Refresh,
				Retry:   record.Retry,
				Expire:  record.Expire,
				Minttl:  record.Minttl,
			}, nil
		}

	case models.RecordTypePTR:
		if qtype == dns.TypePTR {
			return &dns.PTR{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypePTR,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				Ptr: dns.Fqdn(record.Target),
			}, nil
		}

	case models.RecordTypeSRV:
		if qtype == dns.TypeSRV {
			return &dns.SRV{
				Hdr: dns.RR_Header{
					Name:   dns.Fqdn(record.Name),
					Rrtype: dns.TypeSRV,
					Class:  dns.ClassINET,
					Ttl:    record.TTL,
				},
				Priority: uint16(record.Priority),
				Weight:   uint16(record.Weight),
				Port:     uint16(record.Port),
				Target:   dns.Fqdn(record.Target),
			}, nil
		}
	}

	// No matching record type for the query
	return nil, nil
}

// updateTypeStats updates query type statistics
func (s *Server) updateTypeStats(qtype uint16) {
	switch qtype {
	case dns.TypeA:
		s.stats.TypeA++
	case dns.TypeAAAA:
		s.stats.TypeAAAA++
	case dns.TypeCNAME:
		s.stats.TypeCNAME++
	case dns.TypeMX:
		s.stats.TypeMX++
	case dns.TypeTXT:
		s.stats.TypeTXT++
	case dns.TypeNS:
		s.stats.TypeNS++
	case dns.TypeSRV:
		s.stats.TypeSRV++
	case dns.TypeSOA:
		s.stats.TypeSOA++
	case dns.TypePTR:
		s.stats.TypePTR++
	case dns.TypeCAA:
		s.stats.TypeCAA++
	default:
		s.stats.TypeOther++
	}
}
