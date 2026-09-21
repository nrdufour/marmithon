package command

import (
	"os"
	"strings"
	"testing"
)

func TestSoukParseProfile(t *testing.T) {
	page, err := os.ReadFile("testdata/habitue-stam.html")
	if err != nil {
		t.Skip("no testdata snapshot")
	}
	prof := soukParseProfile("stam", string(page))
	if prof == "" {
		t.Fatal("empty profile from valid page")
	}
	for _, want := range []string{
		"stam:",
		"actif 216 jours",
		"première ligne 18 février 2026",
		"dernière 21 septembre 2026",
		"le soir",
		"mots signatures: opais",
		"parle surtout à: CapNemo",
		"trophées: Le Rigolo (n°1)",
	} {
		if !strings.Contains(prof, want) {
			t.Errorf("profile missing %q, got:\n%s", want, prof)
		}
	}
	t.Logf("profile: %s", prof)
}

func TestSoukParseProfileEmpty(t *testing.T) {
	if prof := soukParseProfile("nobody", "<html><body>nothing here</body></html>"); prof != "" {
		t.Errorf("expected empty profile for junk page, got %q", prof)
	}
}
