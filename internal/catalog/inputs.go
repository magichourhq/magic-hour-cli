package catalog

import (
	"fmt"
	"strings"
)

// inferInputs handles API conventions that are not expressed as schema rules.
// It only supplies missing values; explicit choices are validated below.
func (op Operation) inferInputs(values map[string][]string) map[string][]string {
	out := make(map[string][]string, len(values))
	for k, v := range values {
		out[k] = v
	}
	flags := map[string]string{}
	for _, f := range op.Fields {
		flags[strings.Join(f.Path, ".")] = f.Flag
	}
	has := func(path string) bool { v := out[flags[path]]; return len(v) > 0 && v[0] != "" }
	set := func(path, value string) {
		if flag := flags[path]; flag != "" && !has(path) {
			out[flag] = []string{value}
		}
	}
	for _, kind := range []string{"video", "audio"} {
		prefix := "assets." + kind
		if flags[prefix+"_source"] == "" {
			continue
		}
		if has(prefix + "_file_path") {
			set(prefix+"_source", "file")
		} else if has("assets.youtube_url") {
			set(prefix+"_source", "youtube")
		} else if kind == "audio" {
			set(prefix+"_source", "none")
		}
	}
	if has("assets.face_mappings") {
		set("assets.face_swap_mode", "individual-faces")
	}
	if has("style.prompt") {
		set("style.prompt_type", "custom")
	}
	if has("style.points") {
		set("style.selection_mode", "point")
	}
	return out
}

func (op Operation) conditions(body map[string]any) error {
	get := func(path string) any {
		var value any = body
		for _, key := range strings.Split(path, ".") {
			object, ok := value.(map[string]any)
			if !ok {
				return nil
			}
			value = object[key]
		}
		return value
	}
	require := func(path string) error {
		v := get(path)
		if v == nil || v == "" {
			for _, f := range op.Fields {
				if strings.Join(f.Path, ".") == path {
					return fmt.Errorf("missing --%s for the selected options", f.Flag)
				}
			}
			return fmt.Errorf("missing %s", path)
		}
		if items, ok := v.([]any); ok && len(items) == 0 {
			return fmt.Errorf("%s must not be empty", path)
		}
		return nil
	}
	if end, ok := get("end_seconds").(float64); ok {
		start, _ := get("start_seconds").(float64)
		if end <= start {
			return fmt.Errorf("--end must be greater than --start (default 0)")
		}
	}
	for _, kind := range []string{"video", "audio"} {
		source := get("assets." + kind + "_source")
		if source == nil {
			continue
		}
		file := "assets." + kind + "_file_path"
		if get(file) != nil && get("assets.youtube_url") != nil {
			return fmt.Errorf("choose either --%s or --youtube-url", kind)
		}
		switch source {
		case "file":
			if err := require(file); err != nil {
				return err
			}
		case "youtube":
			if err := require("assets.youtube_url"); err != nil {
				return err
			}
		case "none":
			if get(file) != nil || get("assets.youtube_url") != nil {
				return fmt.Errorf("audio-source none cannot include audio input")
			}
		}
	}
	switch op.ID {
	case "faceSwap.createVideo", "faceSwapPhoto.createImage":
		if get("assets.face_swap_mode") == "individual-faces" {
			return require("assets.face_mappings")
		}
		if op.Kind == "video" {
			return require("assets.image_file_path")
		}
		return require("assets.source_file_path")
	case "animation.createVideo":
		if get("style.art_style") == "Custom" {
			if err := require("style.art_style_custom"); err != nil {
				return err
			}
		}
		mode := get("style.prompt_type")
		if mode == "custom" {
			return require("style.prompt")
		}
		if mode == "use_lyrics" || mode == "ai_choose" {
			source := get("assets.audio_source")
			if source != "file" && source != "youtube" {
				return fmt.Errorf("selected prompt type requires --audio or --youtube-url")
			}
		}
	case "videoToVideo.createVideo":
		mode := get("style.prompt_type")
		if mode == "custom" || mode == "append_default" {
			return require("style.prompt")
		}
		if get("style.prompt") != nil {
			return fmt.Errorf("--prompt-type default ignores --prompt; use custom or append_default")
		}
	case "characterReplace.createVideo":
		if get("style.selection_mode") == "point" {
			return require("style.points")
		}
	}
	return nil
}
