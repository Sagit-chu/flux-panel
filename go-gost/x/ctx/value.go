package ctx

import (
	"bytes"
	"context"

	"github.com/go-gost/core/logger"
)

// clientAddrKey saves the client address.
type clientAddrKey struct{}

type ClientAddr string

var (
	keyClientAddr clientAddrKey
)

func ContextWithClientAddr(ctx context.Context, addr ClientAddr) context.Context {
	return context.WithValue(ctx, keyClientAddr, addr)
}

func ClientAddrFromContext(ctx context.Context) ClientAddr {
	v, _ := ctx.Value(keyClientAddr).(ClientAddr)
	return v
}

// sidKey saves the session ID.
type sidKey struct{}
type Sid string

var (
	keySid sidKey
)

func ContextWithSid(ctx context.Context, sid Sid) context.Context {
	return context.WithValue(ctx, keySid, sid)
}

func SidFromContext(ctx context.Context) Sid {
	v, _ := ctx.Value(keySid).(Sid)
	return v
}

// hashKey saves the hash source for Selector.
type hashKey struct{}

type Hash struct {
	Source string
}

var (
	clientHashKey = &hashKey{}
)

func ContextWithHash(ctx context.Context, hash *Hash) context.Context {
	return context.WithValue(ctx, clientHashKey, hash)
}

func HashFromContext(ctx context.Context) *Hash {
	if v, _ := ctx.Value(clientHashKey).(*Hash); v != nil {
		return v
	}
	return nil
}

type clientIDKey struct{}
type ClientID string

var (
	keyClientID = &clientIDKey{}
)

func ContextWithClientID(ctx context.Context, clientID ClientID) context.Context {
	return context.WithValue(ctx, keyClientID, clientID)
}

func ClientIDFromContext(ctx context.Context) ClientID {
	v, _ := ctx.Value(keyClientID).(ClientID)
	return v
}

type bufferKey struct{}

var (
	keyBuffer = &bufferKey{}
)

func ContextWithBuffer(ctx context.Context, buffer *bytes.Buffer) context.Context {
	return context.WithValue(ctx, keyBuffer, buffer)
}

func BufferFromContext(ctx context.Context) *bytes.Buffer {
	v, _ := ctx.Value(keyBuffer).(*bytes.Buffer)
	return v
}

type loggerKey struct{}

var (
	keyLogger = &loggerKey{}
)

func ContextWithLogger(ctx context.Context, log logger.Logger) context.Context {
	return context.WithValue(ctx, keyLogger, log)
}

func LoggerFromContext(ctx context.Context) logger.Logger {
	v, _ := ctx.Value(keyLogger).(logger.Logger)
	return v
}

// excludeNodesKey saves the list of node addresses to exclude during selection.
// This is used for failover retry logic - when a node fails, it gets added to
// the exclude list so the next Select() call will skip it.
type excludeNodesKey struct{}

var (
	keyExcludeNodes = &excludeNodesKey{}
)

// ContextWithExcludeNodes returns a context with the list of node addresses to exclude.
func ContextWithExcludeNodes(ctx context.Context, nodes []string) context.Context {
	return context.WithValue(ctx, keyExcludeNodes, nodes)
}

// ExcludeNodesFromContext returns the list of node addresses to exclude from selection.
func ExcludeNodesFromContext(ctx context.Context) []string {
	v, _ := ctx.Value(keyExcludeNodes).([]string)
	return v
}
