package nbtype

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"":        "",
		"sagoo":   TypeSagoo,
		"SAGOO":   TypeSagoo,
		"xunji":   TypeSagoo,
		" XUNJI ": TypeSagoo,
		"mqtt":    TypeMQTT,
	}

	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported("xunji") {
		t.Fatal("expected xunji to be supported via compatibility")
	}
	if !IsSupported(TypeSagoo) {
		t.Fatal("expected sagoo to be supported")
	}
	if IsSupported("unknown") {
		t.Fatal("expected unknown to be unsupported")
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("xunji"); got != "Sagoo" {
		t.Fatalf("DisplayName(xunji)=%q, want %q", got, "Sagoo")
	}
	if got := DisplayName("pandax"); got != "PandaX" {
		t.Fatalf("DisplayName(pandax)=%q, want %q", got, "PandaX")
	}
}
