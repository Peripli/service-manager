package audit

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gofrs/uuid"
	"net"
	"strings"
	"time"
)

const (
	maxUserAgentLength      = 1024
	userAgentTruncateSuffix = "...TRUNCATED"
)

func NewEventForRequest(r *web.Request) (*Event, error) {
	uuids, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	event := &Event{
		AuditID:          uuids.String(),
		RequestTimestamp: time.Now(),
		RequestObject:    r.Body,
		Verb:             r.Method,
		RequestURI:       r.URL.RequestURI(),
		UserAgent:        userAgent(r),
	}

	ips := SourceIPs(r)
	event.SourceIPs = make([]string, len(ips))
	for i := range ips {
		event.SourceIPs[i] = ips[i].String()
	}
	event.RequestObject = r.Body

	return event, nil
}

func userAgent(request *web.Request) string {
	ua := request.UserAgent()
	if len(ua) > maxUserAgentLength {
		ua = ua[:maxUserAgentLength] + userAgentTruncateSuffix
	}

	return ua
}

func SourceIPs(req *web.Request) []net.IP {
	hdr := req.Header
	// First check the X-Forwarded-For header for requests via proxy.
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	forwardedForIPs := []net.IP{}
	if hdrForwardedFor != "" {
		// X-Forwarded-For can be a csv of IPs in case of multiple proxies.
		// Use the first valid one.
		parts := strings.Split(hdrForwardedFor, ",")
		for _, part := range parts {
			ip := net.ParseIP(strings.TrimSpace(part))
			if ip != nil {
				forwardedForIPs = append(forwardedForIPs, ip)
			}
		}
	}
	if len(forwardedForIPs) > 0 {
		return forwardedForIPs
	}

	// Fallback to Remote Address in request, which will give the correct client IP when there is no proxy.
	// Remote Address in Go's HTTP server is in the form host:port so we need to split that first.
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil {
		if remoteIP := net.ParseIP(host); remoteIP != nil {
			return []net.IP{remoteIP}
		}
	}

	// Fallback if Remote Address was just IP.
	if remoteIP := net.ParseIP(req.RemoteAddr); remoteIP != nil {
		return []net.IP{remoteIP}
	}

	return nil
}
