package version

import "testing"

func TestShort(t *testing.T) {
	tests := []struct {
		version, commit, want string
	}{
		{"v1.2.3", "abc", "v1.2.3"},
		{"dev", "abc1234", "dev·abc1234"},
		{"dev", "", "dev"},
		{"", "", "dev"},
		{"v1.0.0-2-gdeadbeef", "", "v1.0.0-2-gdeadbeef"},
	}
	for _, tt := range tests {
		Version = tt.version
		Commit = tt.commit
		if got := Short(); got != tt.want {
			t.Errorf("Short(%q, %q) = %q, want %q", tt.version, tt.commit, got, tt.want)
		}
	}
}

func TestStringIncludesCommit(t *testing.T) {
	Version = "v1.0.0"
	Commit = "abc1234"
	if got := String(); got != "v1.0.0 (abc1234)" {
		t.Fatalf("String() = %q", got)
	}
}
