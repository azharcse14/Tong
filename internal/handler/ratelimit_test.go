package handler

import "testing"

func TestGetLimiterBurst(t *testing.T) {
	l := getLimiter("1.2.3.4")
	for i := 0; i < 30; i++ {
		if !l.Allow() {
			t.Fatalf("request %d blocked inside burst", i)
		}
	}
	if l.Allow() {
		t.Fatal("31st request should be blocked")
	}
}
