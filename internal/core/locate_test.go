package core

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Passkey login":  "passkey-login",
		"  Audit Log!! ": "audit-log",
		"API v2 / OAuth": "api-v2-oauth",
		"":               "untitled",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q)=%q want %q", in, got, want)
		}
	}
}

func TestMintLineageCollision(t *testing.T) {
	existing := map[string]*Pitch{"2026-07-login": {}}
	got := MintLineage("2026-07", "Login", existing)
	if got != "2026-07-login-2" {
		t.Fatalf("collision suffix wrong: %q", got)
	}
	if !ValidLineageID(got) {
		t.Fatalf("minted id invalid: %q", got)
	}
}

func TestNextADRID(t *testing.T) {
	existing := map[string]*Decision{
		"adr-0001-a": {}, "adr-0007-b": {},
	}
	got := NextADRID("adr", existing, "New Choice")
	if got != "adr-0008-new-choice" {
		t.Fatalf("next adr id wrong: %q", got)
	}
	if !ValidADRID(got) {
		t.Fatalf("minted adr id invalid: %q", got)
	}
}
