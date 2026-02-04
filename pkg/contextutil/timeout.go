package contextutil

import (
	"context"
	"time"
)

// Timeouts par dÃ©faut pour diffÃ©rentes opÃ©rations
const (
	DefaultSSHTimeout        = 30 * time.Second
	DefaultCollectTimeout    = 45 * time.Second
	DefaultDBTimeout         = 5 * time.Second
	DefaultHTTPTimeout       = 60 * time.Second
	DefaultWebSocketTimeout  = 5 * time.Minute
	DefaultShutdownTimeout   = 30 * time.Second
)

// WithTimeout crÃ©e un contexte avec timeout
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}
	return context.WithTimeout(parent, timeout)
}

// WithSSHTimeout crÃ©e un contexte avec timeout SSH
func WithSSHTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DefaultSSHTimeout)
}

// WithCollectTimeout crÃ©e un contexte avec timeout de collection
func WithCollectTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DefaultCollectTimeout)
}

// WithDBTimeout crÃ©e un contexte avec timeout DB
func WithDBTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, DefaultDBTimeout)
}

// WithDeadline crÃ©e un contexte avec deadline absolue
func WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(parent, deadline)
}

// IsTimeout vÃ©rifie si l'erreur est due Ã  un timeout
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return err == context.DeadlineExceeded
}

// IsCanceled vÃ©rifie si l'erreur est due Ã  une annulation
func IsCanceled(err error) bool {
	if err == nil {
		return false
	}
	return err == context.Canceled
}

// IsContextError vÃ©rifie si l'erreur est liÃ©e au contexte
func IsContextError(err error) bool {
	return IsTimeout(err) || IsCanceled(err)
}

// Background retourne un contexte background
func Background() context.Context {
	return context.Background()
}

// TODO retourne un contexte TODO
func TODO() context.Context {
	return context.TODO()
}

// WithValue crÃ©e un contexte avec une valeur
func WithValue(parent context.Context, key, value interface{}) context.Context {
	return context.WithValue(parent, key, value)
}

// GetValue rÃ©cupÃ¨re une valeur du contexte
func GetValue(ctx context.Context, key interface{}) interface{} {
	return ctx.Value(key)
}

// TimeoutConfig contient les configurations de timeout
type TimeoutConfig struct {
	SSH        time.Duration
	Collect    time.Duration
	DB         time.Duration
	HTTP       time.Duration
	WebSocket  time.Duration
	Shutdown   time.Duration
}

// DefaultTimeoutConfig retourne une configuration par dÃ©faut
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		SSH:        DefaultSSHTimeout,
		Collect:    DefaultCollectTimeout,
		DB:         DefaultDBTimeout,
		HTTP:       DefaultHTTPTimeout,
		WebSocket:  DefaultWebSocketTimeout,
		Shutdown:   DefaultShutdownTimeout,
	}
}

// WithSSHTimeoutConfig crÃ©e un contexte avec timeout SSH personnalisÃ©
func WithSSHTimeoutConfig(parent context.Context, config *TimeoutConfig) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, config.SSH)
}

// WithCollectTimeoutConfig crÃ©e un contexte avec timeout de collection personnalisÃ©
func WithCollectTimeoutConfig(parent context.Context, config *TimeoutConfig) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, config.Collect)
}
