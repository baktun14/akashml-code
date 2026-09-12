package cx

import (
	"strings"
	"testing"
)

func TestCheckToken(t *testing.T) {
	akashml := Provider{TokenPrefix: "akml-"}

	tests := []struct {
		name    string
		p       Provider
		token   string
		wantErr bool
	}{
		{name: "a real key passes", p: akashml, token: "akml-EXAMPLEEXAMPLE01", wantErr: false},
		{name: "a password typed at the prompt is rejected", p: akashml, token: "hunter2-not-a-key!", wantErr: true},
		{name: "an Anthropic key pasted by mistake is rejected", p: akashml, token: "sk-ant-api03-xxxx", wantErr: true},
		{name: "an empty token is rejected", p: akashml, token: "", wantErr: true},
		{name: "a provider with no declared prefix accepts anything non-empty", p: Provider{}, token: "whatever", wantErr: false},
		{name: "a provider with no declared prefix still rejects empty", p: Provider{}, token: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckToken(tt.p, tt.token)
			if tt.wantErr && err == nil {
				t.Error("CheckToken accepted a credential it should have refused")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("CheckToken returned %v for a valid credential", err)
			}
		})
	}
}

func TestMaskTokenNeverShowsTheWholeSecret(t *testing.T) {
	token := "akml-EXAMPLEEXAMPLE01"

	masked := MaskToken(token)

	if strings.Contains(masked, token) {
		t.Errorf("MaskToken returned %q, which contains the token in full", masked)
	}
	if !strings.Contains(masked, "akm") {
		t.Errorf("MaskToken returned %q, which shows too little to tell two keys apart", masked)
	}
}

func TestMaskTokenDoesNotIndexPastAShortValue(t *testing.T) {
	if got := MaskToken("abc"); !strings.Contains(got, "3") {
		t.Errorf("MaskToken(short) = %q, want the length reported", got)
	}
	if got := MaskToken(""); !strings.Contains(got, "0") {
		t.Errorf("MaskToken(empty) = %q, want the length reported", got)
	}
}
