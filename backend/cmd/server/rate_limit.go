package main

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var errRateLimitExceeded = errors.New("rate limit exceeded")

type wsRateLimitConfig struct {
	ConnectionRate  rate.Limit
	ConnectionBurst int
	CreateRoomRate  rate.Limit
	CreateRoomBurst int
	JoinRoomRate    rate.Limit
	JoinRoomBurst   int
	MessageRate     rate.Limit
	MessageBurst    int
	VisitorTTL      time.Duration
}

type trackedLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type wsRateLimiters struct {
	config wsRateLimitConfig
	now    func() time.Time

	mu                 sync.Mutex
	lastCleanup        time.Time
	connectionAttempts map[string]*trackedLimiter
	createRoomAttempts map[string]*trackedLimiter
	joinRoomAttempts   map[string]*trackedLimiter
}

func defaultWSRateLimitConfig() wsRateLimitConfig {
	return wsRateLimitConfig{
		ConnectionRate:  rate.Every(2 * time.Second),
		ConnectionBurst: 12,
		CreateRoomRate:  rate.Every(10 * time.Second),
		CreateRoomBurst: 3,
		JoinRoomRate:    rate.Every(2 * time.Second),
		JoinRoomBurst:   10,
		MessageRate:     rate.Limit(20),
		MessageBurst:    60,
		VisitorTTL:      10 * time.Minute,
	}
}

func newWSRateLimiters(config wsRateLimitConfig) *wsRateLimiters {
	config = normalizeWSRateLimitConfig(config)
	now := time.Now
	return &wsRateLimiters{
		config:             config,
		now:                now,
		lastCleanup:        now(),
		connectionAttempts: make(map[string]*trackedLimiter),
		createRoomAttempts: make(map[string]*trackedLimiter),
		joinRoomAttempts:   make(map[string]*trackedLimiter),
	}
}

func normalizeWSRateLimitConfig(config wsRateLimitConfig) wsRateLimitConfig {
	defaults := defaultWSRateLimitConfig()
	if config.ConnectionRate == 0 {
		config.ConnectionRate = defaults.ConnectionRate
	}
	if config.ConnectionBurst <= 0 {
		config.ConnectionBurst = defaults.ConnectionBurst
	}
	if config.CreateRoomRate == 0 {
		config.CreateRoomRate = defaults.CreateRoomRate
	}
	if config.CreateRoomBurst <= 0 {
		config.CreateRoomBurst = defaults.CreateRoomBurst
	}
	if config.JoinRoomRate == 0 {
		config.JoinRoomRate = defaults.JoinRoomRate
	}
	if config.JoinRoomBurst <= 0 {
		config.JoinRoomBurst = defaults.JoinRoomBurst
	}
	if config.MessageRate == 0 {
		config.MessageRate = defaults.MessageRate
	}
	if config.MessageBurst <= 0 {
		config.MessageBurst = defaults.MessageBurst
	}
	if config.VisitorTTL <= 0 {
		config.VisitorTTL = defaults.VisitorTTL
	}
	return config
}

func (l *wsRateLimiters) allowConnectionAttempt(r *http.Request) bool {
	if l == nil {
		return true
	}
	return l.allow(l.connectionAttempts, clientIPFromRequest(r), l.config.ConnectionRate, l.config.ConnectionBurst)
}

func (l *wsRateLimiters) allowCreateRoom(sessionID string) bool {
	if l == nil {
		return true
	}
	return l.allow(l.createRoomAttempts, sessionID, l.config.CreateRoomRate, l.config.CreateRoomBurst)
}

func (l *wsRateLimiters) allowJoinRoom(sessionID string) bool {
	if l == nil {
		return true
	}
	return l.allow(l.joinRoomAttempts, sessionID, l.config.JoinRoomRate, l.config.JoinRoomBurst)
}

func (l *wsRateLimiters) newMessageLimiter() *rate.Limiter {
	if l == nil {
		config := defaultWSRateLimitConfig()
		return rate.NewLimiter(config.MessageRate, config.MessageBurst)
	}
	return rate.NewLimiter(l.config.MessageRate, l.config.MessageBurst)
}

func (l *wsRateLimiters) allow(limiters map[string]*trackedLimiter, key string, limit rate.Limit, burst int) bool {
	if key == "" {
		key = "unknown"
	}

	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupLocked(now)
	tracked := limiters[key]
	if tracked == nil {
		tracked = &trackedLimiter{limiter: rate.NewLimiter(limit, burst)}
		limiters[key] = tracked
	}
	tracked.lastSeen = now

	return tracked.limiter.Allow()
}

func (l *wsRateLimiters) cleanupLocked(now time.Time) {
	if now.Sub(l.lastCleanup) < l.config.VisitorTTL {
		return
	}
	l.deleteStaleLocked(l.connectionAttempts, now)
	l.deleteStaleLocked(l.createRoomAttempts, now)
	l.deleteStaleLocked(l.joinRoomAttempts, now)
	l.lastCleanup = now
}

func (l *wsRateLimiters) deleteStaleLocked(limiters map[string]*trackedLimiter, now time.Time) {
	for key, tracked := range limiters {
		if now.Sub(tracked.lastSeen) >= l.config.VisitorTTL {
			delete(limiters, key)
		}
	}
}

func clientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))
		if value == "" {
			continue
		}
		if header == "X-Forwarded-For" {
			value = strings.TrimSpace(strings.Split(value, ",")[0])
		}
		if ip := net.ParseIP(value); ip != nil {
			return ip.String()
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}
	if ip := net.ParseIP(strings.TrimSpace(r.RemoteAddr)); ip != nil {
		return ip.String()
	}
	return strings.TrimSpace(r.RemoteAddr)
}
