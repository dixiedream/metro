package ui

import (
	"testing"
)

func TestParseColorSpec(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"#ff0000", "#ff0000", true},
		{"#f00", "#ff0000", true},
		{"rgb:ffff/0000/0000", "#ff0000", true},
		{"rgb:ff/00/00", "#ff0000", true},
		{"rgb:1/2/3", "#112233", true},
		{"", "", false},
		{"not-a-color", "", false},
	}
	for _, tc := range tests {
		got, ok := parseColorSpec(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseColorSpec(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestMergeResources(t *testing.T) {
	input := `
! comment
*color0: #000000
URxvt*color1: #cc0000
XTerm*foreground: #eceff1
*background: rgb:11/11/11
*color10: #00ff00
`
	var p Palette
	mergeResources(&p, []byte(input))
	if p.ANSI[0] != "#000000" {
		t.Fatalf("color0: got %q", p.ANSI[0])
	}
	if p.ANSI[1] != "#cc0000" {
		t.Fatalf("color1: got %q", p.ANSI[1])
	}
	if p.ANSI[10] != "#00ff00" {
		t.Fatalf("color10: got %q", p.ANSI[10])
	}
	if p.Foreground != "#eceff1" {
		t.Fatalf("foreground: got %q", p.Foreground)
	}
	if p.Background != "#111111" {
		t.Fatalf("background: got %q", p.Background)
	}
}

func TestPaletteFinalizeFillsANSI(t *testing.T) {
	p := Palette{ANSI: [16]string{"#010101"}}
	out := p.finalize()
	if out.ANSI[0] != "#010101" {
		t.Fatalf("existing color0 preserved")
	}
	if out.ANSI[1] != defaultANSI[1] {
		t.Fatalf("expected default fill for color1")
	}
}

func TestColorDark(t *testing.T) {
	if !colorDark("#1e1e2e") {
		t.Fatal("expected dark")
	}
	if colorDark("#eceff1") {
		t.Fatal("expected light")
	}
}

func TestEnsureReadableDarkBg(t *testing.T) {
	p := Palette{}
	for i := range p.ANSI {
		p.ANSI[i] = defaultANSI[i]
	}
	got := ensureReadable("#000000", "#1e1e2e", true, p)
	if got != defaultANSI[15] {
		t.Fatalf("dark bg + dark fg: got %q want light fallback %q", got, defaultANSI[15])
	}
}
