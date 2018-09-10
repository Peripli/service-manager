package audit

import (
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

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
	return &Event{
		AuditID:          uuids.String(),
		RequestTimestamp: time.Now(),
		RequestObject:    r.Body,
		Verb:             r.Method,
		RequestURI:       r.URL.RequestURI(),
		UserAgent:        userAgent(r),
		SourceIPs:        SourceIPs(r),
	}, nil
}

func userAgent(request *web.Request) string {
	ua := request.UserAgent()
	if len(ua) > maxUserAgentLength {
		ua = ua[:maxUserAgentLength] + userAgentTruncateSuffix
	}

	return ua
}

func SourceIPs(req *web.Request) []string {
	hdrForwardedFor := req.Header.Get("X-Forwarded-For")
	var forwardedForIPs []string
	if hdrForwardedFor != "" {
		parts := strings.Split(hdrForwardedFor, ",")
		for _, part := range parts {
			if ip := net.ParseIP(strings.TrimSpace(part)); ip != nil {
				forwardedForIPs = append(forwardedForIPs, ip.String())
			}
		}
	}
	if len(forwardedForIPs) > 0 {
		return forwardedForIPs
	}

	// Fallback to Remote Address in request, which will give the correct client IP when there is no proxy.
	// Remote Address in Go's HTTP server is in the form host:port so we need to split that first.
	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if remoteIP := net.ParseIP(host); remoteIP != nil {
			return []string{remoteIP.String()}
		}
	}

	// Fallback if Remote Address was just IP.
	if remoteIP := net.ParseIP(req.RemoteAddr); remoteIP != nil {
		return []string{remoteIP.String()}
	}

	return nil
}

func RequestWithEvent(request *web.Request) (*web.Request, *Event, error) {
	event, err := NewEventForRequest(request)
	if err != nil {
		return request, nil, err
	}
	request.Request = request.WithContext(web.ContextWithAuditEvent(request.Context(), event))
	return request, event, nil
}

func LogMetadata(event *Event, key string, value string) {
	if event == nil {
		return
	}
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	if val, exists := event.Metadata[key]; exists {
		logrus.Warnf("Cannot override metadata key %s to %s for audit event %s. Values is already set to %s", key, value, event.AuditID, val)
	}
	event.Metadata[key] = value
}
