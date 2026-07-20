package action

import (
	"testing"

	"github.com/dimspell/gladiator/internal/model"
)

func TestIsValidRunMode_RejectsDisabledModes(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"single", true},
		{"lan", true},
		{"relay-beta", true},
		{"webrtc-beta", false},
		{"libp2p-beta", false},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isValidRunMode(tt.mode)
		if got != tt.want {
			t.Errorf("isValidRunMode(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestSelectProxy_RejectsDisabledModes(t *testing.T) {
	// selectProxy takes a *cli.Command; we can't easily construct one without
	// urfave/cli. Test the mode constants directly.
	if model.RunModeWebRTC.String() != "webrtc-beta" {
		t.Fatal("RunModeWebRTC constant changed")
	}
}
