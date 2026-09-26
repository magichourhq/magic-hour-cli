package catalog

import (
	"strings"
	"testing"
)

func TestFaceSwapNestedMappings(t *testing.T) {
	var op Operation
	for _, candidate := range Operations {
		if candidate.Group == "image" && candidate.Name == "face-swap" {
			op = candidate
			break
		}
	}
	if op.ID == "" {
		t.Fatal("image face-swap command missing")
	}
	values := map[string][]string{
		"target":        {"target.png"},
		"face-mappings": {`[{"original_face":"api-assets/old.png","new_face":"new.png"}]`},
	}
	body, err := op.Body(values)
	if err != nil {
		t.Fatal(err)
	}
	assets := body["assets"].(map[string]any)
	if assets["face_swap_mode"] != "individual-faces" {
		t.Fatalf("face mappings did not select individual mode: %v", assets)
	}
	mappings := assets["face_mappings"].([]any)
	if mappings[0].(map[string]any)["new_face"] != "new.png" {
		t.Fatalf("nested mapping lost: %v", mappings)
	}
	for _, tc := range []struct {
		mapping, want string
	}{
		{`[{"original_face":"api-assets/old.png"}]`, "new_face"},
		{`[{"original_face":"api-assets/old.png","new_face":5}]`, "expected a string"},
	} {
		values["face-mappings"] = []string{tc.mapping}
		if _, err := op.Body(values); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("got %v, want error containing %q", err, tc.want)
		}
	}
}

func TestClipEndMustFollowStart(t *testing.T) {
	for _, op := range Operations {
		if op.Group == "video" && op.Name == "replace-character" {
			_, err := op.Body(map[string][]string{
				"image": {"character.png"}, "video": {"scene.mp4"}, "start": {"5"}, "end": {"5"},
			})
			if err == nil || !strings.Contains(err.Error(), "--end must be greater than --start") {
				t.Fatalf("got %v, want invalid clip range", err)
			}
			return
		}
	}
	t.Fatal("video replace-character command missing")
}
