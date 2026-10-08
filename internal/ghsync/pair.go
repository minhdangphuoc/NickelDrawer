package ghsync

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"koboapps/internal/i18n"
)

// PairOutcome says how a pairing session ended.
type PairOutcome string

const (
	PairPaired  PairOutcome = "paired"
	PairLocked  PairOutcome = "locked"
	PairExpired PairOutcome = "expired"
	PairStopped PairOutcome = "stopped"
)

const (
	// DefaultPairPort is where the pairing page listens on the Kobo's LAN address.
	DefaultPairPort = 8765
	// DefaultPairLifetime bounds how long the page stays reachable.
	DefaultPairLifetime = 10 * time.Minute

	maxCodeAttempts     = 5
	maxFormBytes        = 8 * 1024
	shutdownGracePeriod = 5 * time.Second
	codeDigits          = 6
)

// PairOptions configure a pairing session.
type PairOptions struct {
	Paths    Paths
	APIBase  string
	IP       string // LAN address to bind; empty = detect
	Port     int
	Lifetime time.Duration
	Locale   string
	Messages *i18n.Bundle
	Log      *Logger
	Out      io.Writer // announcements for the Kobo screen
	Backoff  time.Duration

	// Code and Now are test seams; zero values mean a random code and the real clock.
	Code string
	Now  func() time.Time
}

// Pair serves the pairing page until it succeeds, locks, expires or ctx ends.
func Pair(ctx context.Context, options PairOptions) (PairOutcome, error) {
	bindIP, err := resolvePairIP(options.IP)
	if err != nil {
		return "", err
	}
	session, err := newPairSession(options)
	if err != nil {
		return "", err
	}
	address := net.JoinHostPort(bindIP, strconv.Itoa(options.Port))
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		return "", i18n.NewError("msg.err_pair_listen", "address", address).WithCause(err)
	}
	server := &http.Server{
		Handler:           session,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	go server.Serve(listener)
	session.announce(listener.Addr().String())

	outcome := session.wait(ctx)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	session.options.Log.Printf("pairing ended: %s", outcome)
	return outcome, shutdownErr
}

// resolvePairIP returns the address to bind. It never returns a wildcard address,
// so the page is not reachable on other interfaces.
func resolvePairIP(requested string) (string, error) {
	if requested == "" {
		return DetectLANIP()
	}
	ip := net.ParseIP(requested)
	if ip == nil || ip.To4() == nil || ip.IsUnspecified() {
		return "", i18n.NewError("msg.err_pair_ip_invalid", "ip", requested)
	}
	return requested, nil
}

// DetectLANIP returns the first private IPv4 address of an up, non-loopback interface.
func DetectLANIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", i18n.NewError("msg.err_pair_no_ip").WithCause(err)
	}
	for _, networkInterface := range interfaces {
		if networkInterface.Flags&net.FlagUp == 0 || networkInterface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			network, isNetwork := address.(*net.IPNet)
			if isNetwork && network.IP.To4() != nil && network.IP.IsPrivate() {
				return network.IP.String(), nil
			}
		}
	}
	return "", i18n.NewError("msg.err_pair_no_ip")
}

// pairSession is the state behind the pairing page.
type pairSession struct {
	options PairOptions
	code    string
	expires time.Time
	now     func() time.Time

	mutex    sync.Mutex
	attempts int
	finished bool
	outcome  chan PairOutcome
}

func newPairSession(options PairOptions) (*pairSession, error) {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Lifetime <= 0 {
		options.Lifetime = DefaultPairLifetime
	}
	if options.Backoff <= 0 {
		options.Backoff = time.Second
	}
	if options.Log == nil {
		options.Log = NewLogger(io.Discard, "")
	}
	if options.Out == nil {
		options.Out = io.Discard
	}
	code := options.Code
	if code == "" {
		var err error
		if code, err = newPairCode(); err != nil {
			return nil, i18n.NewError("msg.err_pair_code").WithCause(err)
		}
	}
	return &pairSession{
		options: options,
		code:    code,
		now:     options.Now,
		expires: options.Now().Add(options.Lifetime),
		outcome: make(chan PairOutcome, 1),
	}, nil
}

// newPairCode draws a uniform 6-digit code from the system random source.
func newPairCode() (string, error) {
	limit := big.NewInt(1)
	for range codeDigits {
		limit.Mul(limit, big.NewInt(10))
	}
	number, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", codeDigits, number), nil
}

// announce prints what the user needs on the Kobo screen: where to go and the code.
func (session *pairSession) announce(address string) {
	messages := session.options.Messages
	minutes := strconv.Itoa(int(session.options.Lifetime / time.Minute))
	fmt.Fprintln(session.options.Out, messages.Message("msg.pair_url", i18n.Params("url", "http://"+address)))
	fmt.Fprintln(session.options.Out, messages.Message("msg.pair_code", i18n.Params("code", session.code)))
	fmt.Fprintln(session.options.Out, messages.Message("msg.pair_valid", i18n.Params("minutes", minutes)))
	session.options.Log.Printf("pairing started on %s", address)
}

func (session *pairSession) wait(ctx context.Context) PairOutcome {
	timer := time.NewTimer(session.expires.Sub(session.now()))
	defer timer.Stop()
	select {
	case outcome := <-session.outcome:
		return outcome
	case <-timer.C:
		return PairExpired
	case <-ctx.Done():
		return PairStopped
	}
}

// finish records the outcome once; later calls are ignored.
func (session *pairSession) finish(outcome PairOutcome) {
	if session.finished {
		return
	}
	session.finished = true
	session.outcome <- outcome
}

func (session *pairSession) isExpired() bool { return !session.now().Before(session.expires) }

// codeMatches compares in constant time so timing does not leak digits.
func (session *pairSession) codeMatches(given string) bool {
	return constantTimeEqual(session.code, given)
}
