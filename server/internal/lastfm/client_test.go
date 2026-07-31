package lastfm

import "testing"

func TestSign(t *testing.T) {
	params := map[string]string{
		"method":  "auth.getSession",
		"api_key": "xxxxxxxx",
		"token":   "xxxxxxx",
		"format":  "json",
	}
	got := Sign(params, "mysecret")
	want := "68afb32bee072407a63b6c41f3e1e2b4"
	if got != want {
		t.Fatalf("sign = %q, want %q", got, want)
	}
}

func TestPrimaryArtist(t *testing.T) {
	if got := PrimaryArtist("JAŸ-Z · Kanye West"); got != "JAŸ-Z" {
		t.Fatalf("got %q", got)
	}
	if got := PrimaryArtist("Skrillex"); got != "Skrillex" {
		t.Fatalf("got %q", got)
	}
}
