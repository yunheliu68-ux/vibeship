package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillItem represents a single discovered skill.
type SkillItem struct {
	Name     string
	Active   bool
	Category string // e.g. "superpowers", "lark", "figma"
}

// PluginItem represents a single discovered plugin.
type PluginItem struct {
	Name   string
	Active bool
}

// Registry holds the full list of discovered skills and plugins.
type Registry struct {
	Skills  []SkillItem
	Plugins []PluginItem
}

// LoadSkillsAndPlugins scans the Claude Code home directory and returns a
// Registry of installed skills and plugins. A nil error does not mean the
// registry is non-empty — empty directories simply produce an empty registry.
func LoadSkillsAndPlugins(homeDir string) (*Registry, error) {
	r := &Registry{}

	// Read settings.json for enabled plugins
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return r, nil // no settings, return empty
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return r, nil
	}

	// Track which plugins are enabled
	enabledPlugins := make(map[string]bool)
	if ep, ok := settings["enabledPlugins"].(map[string]interface{}); ok {
		for name, enabled := range ep {
			if v, ok := enabled.(bool); ok && v {
				enabledPlugins[name] = true
			}
		}
	}

	// Scan skills directory
	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	skillEntries, _ := os.ReadDir(skillsDir)
	for _, entry := range skillEntries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Check subdirectories for plugin-namespaced skills
		subEntries, err := os.ReadDir(filepath.Join(skillsDir, name))
		if err != nil {
			continue
		}
		for _, sub := range subEntries {
			if sub.IsDir() {
				fullName := name + ":" + sub.Name()
				r.Skills = append(r.Skills, SkillItem{
					Name:     fullName,
					Category: name,
					Active:   false, // updated by transcript events
				})
			}
		}
	}

	// Scan plugins directory for marketplace names
	pluginsDir := filepath.Join(homeDir, ".claude", "plugins")
	pluginEntries, _ := os.ReadDir(pluginsDir)
	for _, entry := range pluginEntries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Try to read plugin metadata
		manifestPath := filepath.Join(pluginsDir, name, "manifest.json")
		var pluginName string
		if manifestData, err := os.ReadFile(manifestPath); err == nil {
			var manifest struct {
				Name string `json:"name"`
			}
			if json.Unmarshal(manifestData, &manifest) == nil && manifest.Name != "" {
				pluginName = manifest.Name
			}
		}
		if pluginName == "" {
			pluginName = name
		}

		r.Plugins = append(r.Plugins, PluginItem{
			Name:   pluginName,
			Active: enabledPlugins[pluginName],
		})
	}

	// Sort skills by category then name
	sort.Slice(r.Skills, func(i, j int) bool {
		if r.Skills[i].Category != r.Skills[j].Category {
			return r.Skills[i].Category < r.Skills[j].Category
		}
		return r.Skills[i].Name < r.Skills[j].Name
	})

	return r, nil
}

// ParseScopeFile reads SCOPE.md or PRD.md from a project directory
// and returns extracted sections.
func ParseScopeFile(projectDir string) (*Scope, error) {
	for _, name := range []string{"SCOPE.md", "PRD.md"} {
		data, err := os.ReadFile(filepath.Join(projectDir, name))
		if err != nil {
			continue
		}
		return parseScopeMarkdown(string(data)), nil
	}
	return nil, nil // no scope file found, that's OK
}

// Scope holds the sections extracted from a SCOPE.md / PRD.md file.
type Scope struct {
	Goals        []string
	Files        []string
	OutOfScope   []string
	DevelopOrder []string
}

func parseScopeMarkdown(content string) *Scope {
	s := &Scope{}
	currentSection := ""
	for _, line := range splitLines(content) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			switch strings.ToLower(strings.TrimPrefix(line, "## ")) {
			case "goals":
				currentSection = "goals"
			case "files":
				currentSection = "files"
			case "out of scope":
				currentSection = "out"
			case "develop order":
				currentSection = "order"
			default:
				currentSection = ""
			}
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			item := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			switch currentSection {
			case "goals":
				s.Goals = append(s.Goals, item)
			case "files":
				s.Files = append(s.Files, item)
			case "out":
				s.OutOfScope = append(s.OutOfScope, item)
			case "order":
				s.DevelopOrder = append(s.DevelopOrder, item)
			}
		}
	}
	return s
}

func splitLines(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' })
}

// Ensure yaml import is used (referenced at package level).
var _ = yaml.Marshal
