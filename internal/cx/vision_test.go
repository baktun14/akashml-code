package cx

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestProbeVisionClassifiesWhatAModelDoesWithAnImage(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    VisionSupport
	}{
		{
			name: "names the colour, so it looked",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"content":[{"type":"text","text":"Red"}]}`))
			},
			want: VisionReads,
		},
		{
			name: "answers without the colour, so the image was dropped",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"content":[{"type":"text","text":"I see it."}]}`))
			},
			want: VisionIgnores,
		},
		{
			name:    "empty reply is also a drop",
			handler: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"content":[]}`)) },
			want:    VisionIgnores,
		},
		{
			name: "a refusal is a rejection",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"type":"error","error":{"message":"Model only supports text input"}}`))
			},
			want: VisionRejects,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			got, err := ProbeVision(Provider{BaseURL: server.URL}, "akml-token", "some-model")
			if err != nil {
				t.Fatalf("ProbeVision returned %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProbeVisionSendsARealImage(t *testing.T) {
	raw, err := base64.StdEncoding.DecodeString(probeImage)
	if err != nil {
		t.Fatalf("the embedded probe image is not valid base64: %v", err)
	}
	if string(raw[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatal("the embedded probe image is not a PNG, so a model would have nothing to look at")
	}
}

func TestCapabilitiesSurviveARoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capabilities.json")

	caps := LoadCapabilities(path)
	if got := caps.VisionFor("akashml", "anything"); got != VisionUnknown {
		t.Errorf("an unprobed model reported %q, want unknown", got)
	}

	caps.SetVision("akashml", "a-model", VisionRejects)
	if err := SaveCapabilities(path, caps); err != nil {
		t.Fatalf("SaveCapabilities returned %v", err)
	}

	if got := LoadCapabilities(path).VisionFor("akashml", "a-model"); got != VisionRejects {
		t.Errorf("after a reload got %q, want %q", got, VisionRejects)
	}
}

func TestLoadCapabilitiesTreatsABrokenCacheAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capabilities.json")
	write(t, path, "{not json")

	caps := LoadCapabilities(path)

	if caps.Vision == nil {
		t.Fatal("a corrupt cache left a nil map, which would panic on the next write")
	}
	caps.SetVision("p", "m", VisionReads)
}
