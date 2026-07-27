package realtime

import (
	"testing"
)

func TestCompareSubscriptionsArePairScoped(t *testing.T) {
	hub := NewHub()
	client := &client{send: make(chan []byte, 1), comparePairs: map[string]struct{}{}}
	hub.handleSubscription(client, []byte(`{"channel":"compare","action":"subscribe","pairs":["BTC_USDT","ETH_USDT"]}`))
	if _, ok := client.comparePairs["BTC_USDT"]; !ok {
		t.Fatal("subscribe did not add BTC_USDT")
	}
	hub.handleSubscription(client, []byte(`{"channel":"compare","action":"unsubscribe","pairs":["BTC_USDT"]}`))
	if _, ok := client.comparePairs["BTC_USDT"]; ok {
		t.Fatal("unsubscribe did not remove BTC_USDT")
	}
	if _, ok := client.comparePairs["ETH_USDT"]; !ok {
		t.Fatal("unsubscribe removed other pair")
	}
}
