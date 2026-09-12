package cx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// VisionSupport is what a model does when a request carries an image. The
// middle case matters most: a model that accepts the request and silently drops
// the picture gives no error to notice.
type VisionSupport string

const (
	VisionUnknown VisionSupport = ""
	VisionReads   VisionSupport = "reads"
	VisionIgnores VisionSupport = "ignores"
	VisionRejects VisionSupport = "rejects"
)

func (v VisionSupport) Describe() string {
	switch v {
	case VisionReads:
		return "reads images"
	case VisionIgnores:
		return "drops images"
	case VisionRejects:
		return "REJECTS images"
	}
	return ""
}

// probeColour is the answer a model that actually looked will give. A solid
// field of one colour leaves little room for a differing description.
const probeColour = "red"

// probeImage is a 32x32 solid red PNG.
const probeImage = "iVBORw0KGgoAAAANSUhEUgAAACAAAAAgCAIAAAD8GO2jAAAAKElEQVR4nO3NsQ0AAA" +
	"zCMP5/un0CNkuZ41wybXsHAAAAAAAAAAAAxR4yw/wuPL6QkAAAAABJRU5ErkJggg=="

// ProbeVision asks a model to name the colour of an image. The gateway's model
// list says nothing about capabilities, so the only honest way to know is to
// send one and look at what comes back.
func ProbeVision(p Provider, token, model string) (VisionSupport, error) {
	body, err := json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": 64,
		"messages": []any{map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "image", "source": map[string]any{
					"type": "base64", "media_type": "image/png", "data": probeImage,
				}},
				map[string]any{"type": "text", "text": "What colour is this image? Answer in one word."},
			},
		}},
	})
	if err != nil {
		return VisionUnknown, err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(p.BaseURL, "/")+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return VisionUnknown, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return VisionUnknown, fmt.Errorf("probing %s: %w", model, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return VisionRejects, nil
	}

	var answer struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		return VisionUnknown, fmt.Errorf("reading the probe reply for %s: %w", model, err)
	}

	var text string
	for _, block := range answer.Content {
		text += block.Text
	}
	if strings.Contains(strings.ToLower(text), probeColour) {
		return VisionReads, nil
	}
	return VisionIgnores, nil
}
