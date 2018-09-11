/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package audit

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/Peripli/service-manager/pkg/web"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

const auditKey = iota

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
		UserAgent:        r.UserAgent(),
		SourceIPs:        SourceIPs(r),
	}, nil
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
	request.Request = request.WithContext(ContextWithEvent(request.Context(), event))
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

func ContextWithEvent(ctx context.Context, event *Event) context.Context {
	return context.WithValue(ctx, auditKey, event)
}

func EventFromContext(ctx context.Context) *Event {
	event, _ := ctx.Value(auditKey).(*Event)
	return event
}
