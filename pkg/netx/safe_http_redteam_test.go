package netx

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRedTeamSafeTransportBlocksRestrictedLiteralTargets(t *testing.T) {
	previous := AllowPrivateIPsInTest
	AllowPrivateIPsInTest = false
	t.Cleanup(func() { AllowPrivateIPsInTest = previous })

	transport := NewSafeTransport(false).(*http.Transport)
	defer transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for _, target := range []string{
		"127.0.0.1:1",
		"[::1]:1",
		"10.0.0.1:1",
		"169.254.169.254:1",
		"[64:ff9b::7f00:1]:1",
	} {
		t.Run(target, func(t *testing.T) {
			conn, err := transport.DialContext(ctx, "tcp", target)
			if conn != nil {
				conn.Close()
				t.Fatal("restricted literal target unexpectedly connected")
			}
			if err == nil || !strings.Contains(err.Error(), "private/restricted IP blocked") {
				t.Fatal("restricted literal target was not rejected by the IP policy")
			}
		})
	}

	client := NewSafeHTTPClient(time.Second)
	redirect, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1/", nil)
	if err != nil {
		t.Fatal("cannot construct local redirect request")
	}
	if err := client.CheckRedirect(redirect, []*http.Request{redirect}); err == nil {
		t.Fatal("restricted redirect target unexpectedly accepted")
	}
}
