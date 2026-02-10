package nbtype

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"":         "",
		"sagoo":    TypeSagoo,
		"SAGOO":    TypeSagoo,
		"mqtt":     TypeMQTT,
		" PANDAX ": TypePandaX,
	}

	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported(TypeSagoo) {
		t.Fatal("expected sagoo to be supported")
	}
	if !IsSupported(TypeMQTT) {
		t.Fatal("expected mqtt to be supported")
	}
	if IsSupported("xunji") {
		t.Fatal("expected xunji to be unsupported")
	}
	if IsSupported("unknown") {
		t.Fatal("expected unknown to be unsupported")
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("sagoo"); got != "Sagoo" {
		t.Fatalf("DisplayName(sagoo)=%q, want %q", got, "Sagoo")
	}
	if got := DisplayName("pandax"); got != "PandaX" {
		t.Fatalf("DisplayName(pandax)=%q, want %q", got, "PandaX")
	}
}
