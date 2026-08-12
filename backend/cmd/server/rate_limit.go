package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var errRateLimitExceeded = errors.New("rate limit exceeded")

const maxUnauthenticatedConnectionsPerIP = 32

type wsRateLimitConfig struct {
	ConnectionRate  rate.Limit
	ConnectionBurst int
	CreateRoomRate  rate.Limit
	CreateRoomBurst int
	JoinRoomRate    rate.Limit
	JoinRoomBurst   int
	MessageRate     rate.Limit
	MessageBurst    int
	HTTPRate        rate.Limit
	HTTPBurst       int
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
	httpRequests       map[string]*trackedLimiter
	activeConnections  map[string]int
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
		HTTPRate:        rate.Limit(20),
		HTTPBurst:       120,
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
		httpRequests:       make(map[string]*trackedLimiter),
		activeConnections:  make(map[string]int),
	}
}

func (l *wsRateLimiters) beginConnection(r *http.Request) (func(), bool) {
	if l == nil {
		return func() {}, true
	}
	if !l.allowConnectionAttempt(r) {
		return nil, false
	}
	key := clientIPFromRequest(r)
	if key == "" {
		key = "unknown"
	}
	l.mu.Lock()
	if l.activeConnections[key] >= maxUnauthenticatedConnectionsPerIP {
		l.mu.Unlock()
		return nil, false
	}
	l.activeConnections[key]++
	l.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			l.activeConnections[key]--
			if l.activeConnections[key] <= 0 {
				delete(l.activeConnections, key)
			}
			l.mu.Unlock()
		})
	}, true
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
	if config.HTTPRate == 0 {
		config.HTTPRate = defaults.HTTPRate
	}
	if config.HTTPBurst <= 0 {
		config.HTTPBurst = defaults.HTTPBurst
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

func (l *wsRateLimiters) allowHTTPRequest(r *http.Request) bool {
	if l == nil {
		return true
	}
	return l.allow(l.httpRequests, clientIPFromRequest(r), l.config.HTTPRate, l.config.HTTPBurst)
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
	l.deleteStaleLocked(l.httpRequests, now)
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

	peer := remoteHost(r.RemoteAddr)
	if proxyHeadersExplicitlyTrusted() {
		if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
			return ip.String()
		}
		return peer
	}
	if !trustedProxyIP(peer) {
		return peer
	}

	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		hops := strings.Split(forwarded, ",")
		for index := len(hops) - 1; index >= 0; index-- {
			if ip := net.ParseIP(strings.TrimSpace(hops[index])); ip != nil && !trustedProxyIP(ip.String()) {
				return ip.String()
			}
		}
	}
	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}

	return peer
}

func proxyHeadersExplicitlyTrusted() bool {
	trusted, err := strconv.ParseBool(strings.TrimSpace(os.Getenv("TRUST_PROXY_HEADERS")))
	return err == nil && trusted
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}
	if ip := net.ParseIP(strings.TrimSpace(remoteAddr)); ip != nil {
		return ip.String()
	}
	return strings.TrimSpace(remoteAddr)
}

func trustedProxyIP(peer string) bool {
	ip := net.ParseIP(peer)
	if ip == nil {
		return false
	}
	for _, rawCIDR := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(rawCIDR))
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}
