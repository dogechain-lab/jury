package jsonrpc

import (
	"net"
	"testing"

	"github.com/dogechain-lab/dogechain/helper/tests"

	"github.com/hashicorp/go-hclog"
)

func TestHTTPServer(t *testing.T) {
	store := newMockStore()
	port, portErr := tests.GetFreePort()

	if portErr != nil {
		t.Fatalf("Unable to fetch free port, %v", portErr)
	}

	config := &Config{
		Store: store,
		Addr:  &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: port},
	}
	_, err := NewJSONRPC(hclog.NewNullLogger(), config)

	if err != nil {
		t.Fatal(err)
	}
}
