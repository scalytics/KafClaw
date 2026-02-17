package cliconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/KafClaw/KafClaw/internal/config"
)

type pathToken struct {
	key   string
	index *int
}

// Get returns the effective config value at a path (dot + bracket notation).
func Get(path string) (any, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	var m map[string]any
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	tokens, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	val, ok := getAtPath(m, tokens)
	if !ok {
		return nil, fmt.Errorf("path not found: %s", path)
	}
	return val, nil
}

// Set writes a value at path into the root config file.
// Value can be JSON or plain string.
func Set(path, rawValue string) error {
	cfgMap, cfgPath, err := loadFileConfigMap()
	if err != nil {
		return err
	}
	tokens, err := parsePath(path)
	if err != nil {
		return err
	}
	root, err := setAtPath(cfgMap, tokens, parseValue(rawValue))
	if err != nil {
		return err
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid config root after set")
	}
	return saveFileConfigMap(cfgPath, rootMap)
}

// Unset removes a value at path from the root config file.
func Unset(path string) error {
	cfgMap, cfgPath, err := loadFileConfigMap()
	if err != nil {
		return err
	}
	tokens, err := parsePath(path)
	if err != nil {
		return err
	}
	root, ok, err := unsetAtPath(cfgMap, tokens)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("path not found: %s", path)
	}
	rootMap, rootOK := root.(map[string]any)
	if !rootOK {
		return fmt.Errorf("invalid config root after unset")
	}
	return saveFileConfigMap(cfgPath, rootMap)
}

func loadFileConfigMap() (map[string]any, string, error) {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, cfgPath, nil
		}
		return nil, "", err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, "", err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, cfgPath, nil
}

func saveFileConfigMap(cfgPath string, m map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0o600)
}

func parsePath(path string) ([]pathToken, error) {
	s := strings.TrimSpace(path)
	if s == "" {
		return nil, fmt.Errorf("path is empty")
	}
	var out []pathToken
	i := 0
	for i < len(s) {
		if s[i] == '.' {
			i++
			continue
		}

		start := i
		for i < len(s) && s[i] != '.' && s[i] != '[' {
			i++
		}
		if i > start {
			k := strings.TrimSpace(s[start:i])
			if k != "" {
				out = append(out, pathToken{key: k})
			}
		}

		for i < len(s) && s[i] == '[' {
			i++
			idxStart := i
			for i < len(s) && s[i] != ']' {
				i++
			}
			if i >= len(s) || s[i] != ']' {
				return nil, fmt.Errorf("invalid path: missing closing ] in %q", path)
			}
			rawIdx := strings.TrimSpace(s[idxStart:i])
			idx, err := strconv.Atoi(rawIdx)
			if err != nil || idx < 0 {
				return nil, fmt.Errorf("invalid array index %q in %q", rawIdx, path)
			}
			i++
			out = append(out, pathToken{index: &idx})
		}

		if i < len(s) && s[i] == '.' {
			i++
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("path is empty")
	}
	return out, nil
}

func parseValue(raw string) any {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err == nil {
		return v
	}
	return raw
}

func getAtPath(root map[string]any, path []pathToken) (any, bool) {
	if len(path) == 0 {
		return root, true
	}
	cur := any(root)
	for _, tok := range path {
		if tok.index != nil {
			arr, ok := cur.([]any)
			if !ok || *tok.index < 0 || *tok.index >= len(arr) {
				return nil, false
			}
			cur = arr[*tok.index]
			continue
		}
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := obj[tok.key]
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func setAtPath(root map[string]any, path []pathToken, value any) (any, error) {
	if len(path) == 0 {
		return root, fmt.Errorf("path is empty")
	}
	return setNode(root, path, value), nil
}

func setNode(node any, path []pathToken, value any) any {
	if len(path) == 0 {
		return value
	}
	tok := path[0]
	rest := path[1:]

	if tok.index != nil {
		var arr []any
		existing, ok := node.([]any)
		if ok {
			arr = existing
		}
		for len(arr) <= *tok.index {
			arr = append(arr, nil)
		}
		arr[*tok.index] = setNode(arr[*tok.index], rest, value)
		return arr
	}

	obj, ok := node.(map[string]any)
	if !ok {
		obj = map[string]any{}
	}
	obj[tok.key] = setNode(obj[tok.key], rest, value)
	return obj
}

func unsetAtPath(root map[string]any, path []pathToken) (any, bool, error) {
	if len(path) == 0 {
		return root, false, fmt.Errorf("path is empty")
	}
	return unsetNode(root, path)
}

func unsetNode(node any, path []pathToken) (any, bool, error) {
	if len(path) == 0 {
		return node, false, nil
	}
	tok := path[0]
	rest := path[1:]

	if len(rest) == 0 {
		if tok.index != nil {
			arr, ok := node.([]any)
			if !ok || *tok.index < 0 || *tok.index >= len(arr) {
				return node, false, nil
			}
			arr = append(arr[:*tok.index], arr[*tok.index+1:]...)
			return arr, true, nil
		}
		obj, ok := node.(map[string]any)
		if !ok {
			return node, false, nil
		}
		if _, ok := obj[tok.key]; !ok {
			return node, false, nil
		}
		delete(obj, tok.key)
		return obj, true, nil
	}

	if tok.index != nil {
		arr, ok := node.([]any)
		if !ok || *tok.index < 0 || *tok.index >= len(arr) {
			return node, false, nil
		}
		newChild, changed, err := unsetNode(arr[*tok.index], rest)
		if err != nil {
			return node, false, err
		}
		if !changed {
			return node, false, nil
		}
		arr[*tok.index] = newChild
		return arr, true, nil
	}

	obj, ok := node.(map[string]any)
	if !ok {
		return node, false, nil
	}
	child, ok := obj[tok.key]
	if !ok {
		return node, false, nil
	}
	newChild, changed, err := unsetNode(child, rest)
	if err != nil {
		return node, false, err
	}
	if !changed {
		return node, false, nil
	}
	obj[tok.key] = newChild
	return obj, true, nil
}
