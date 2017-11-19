package main


import (
	"net/http"
	"net/httptest"
	"testing"
)


func TestGoLink(t *testing.T) {
	var g GoLink
	g.Target = "http://go/link"
	if g.Target != "http://go/link" {
		t.Error("Expected http://go/link, got: %v", g.Target)
	}
}


func TestParseInboundPath(t *testing.T) {
	name, args := ParseInboundPath("/wiki")
	if name != "wiki" {
		t.Error("Expected 'wiki', got: %v", name)
	}
	if args != "" {
		t.Error("Expected '', got: %v", args)
	}
}
