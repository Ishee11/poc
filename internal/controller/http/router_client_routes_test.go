package http

import "testing"

func TestAccountScreenUsesNonAPIClientRoute(t *testing.T) {
	if !isClientRoute("/profile") {
		t.Fatal("profile must be served as a client route")
	}
	if isClientRoute("/account") {
		t.Fatal("account is an API route and must not be treated as a client route")
	}
}
