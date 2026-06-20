package rules

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/store"
)

// RecommendSkill returns a skill name to suggest based on recent activity.
func RecommendSkill(events []store.TranscriptEvent) string {
	// Look at last 5 minutes of events
	cutoff := time.Now().Add(-5 * time.Minute)
	hasEdit := false
	hasBashError := false
	hasNewBranch := false

	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		switch e.EventType {
		case "tool_call":
			if e.Name == "Write" || e.Name == "Edit" {
				hasEdit = true
			}
			if e.Name == "Bash" && strings.Contains(strings.ToLower(e.Detail), "error") {
				hasBashError = true
			}
		}
	}

	if hasEdit {
		return "code-review"
	}
	if hasBashError {
		return "systematic-debugging"
	}
	if hasNewBranch {
		return "writing-plans"
	}
	return ""
}

// GenerateCheckQuestions produces scope/thinking questions based on recent activity.
func GenerateCheckQuestions(scope *config.Scope, events []store.TranscriptEvent) []string {
	var questions []string
	cutoff := time.Now().Add(-10 * time.Minute)

	if scope != nil {
		// Check if recent file edits are within scope
		var outOfScopeFiles []string
		var inScopeFiles []string
		for _, e := range events {
			if e.Timestamp.Before(cutoff) {
				continue
			}
			if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
				if e.Detail == "" {
					continue
				}
				if isInScope(e.Detail, scope.Files) {
					inScopeFiles = append(inScopeFiles, e.Detail)
				} else {
					outOfScopeFiles = append(outOfScopeFiles, e.Detail)
				}
			}
		}

		if len(outOfScopeFiles) > 0 {
			questions = append(questions,
				"⚠️ 这些文件不在 SCOPE.md 里：\n   "+strings.Join(outOfScopeFiles, ", ")+"\n   确定要继续改吗？")
		}

		// Check if writing code without schema changes
		hasWrite := false
		hasSchema := false
		for _, e := range events {
			if e.Timestamp.Before(cutoff) {
				continue
			}
			if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
				hasWrite = true
			}
			if e.EventType == "tool_call" && e.Name == "Write" {
				if strings.Contains(e.Detail, "migration") ||
					strings.Contains(e.Detail, "schema") ||
					strings.Contains(e.Detail, ".sql") {
					hasSchema = true
				}
			}
		}
		if hasWrite && !hasSchema && len(scope.DevelopOrder) > 0 {
			questions = append(questions,
				"💡 先定义数据结构再写代码。Scope 里的开发顺序：\n   "+strings.Join(scope.DevelopOrder, " → "))
		}

		// Check frontend/backend coordination
		hasFrontend := false
		hasBackend := false
		for _, e := range events {
			if e.Timestamp.Before(cutoff) {
				continue
			}
			if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
				ext := strings.ToLower(filepath.Ext(e.Detail))
				dir := filepath.Dir(e.Detail)
				if ext == ".tsx" || ext == ".jsx" || ext == ".vue" || strings.Contains(dir, "frontend") || strings.Contains(dir, "components") {
					hasFrontend = true
				}
				if ext == ".go" || ext == ".py" || ext == ".rs" || strings.Contains(dir, "backend") || strings.Contains(dir, "api") {
					hasBackend = true
				}
			}
		}
		if hasFrontend && !hasBackend {
			questions = append(questions,
				"💡 只改了前端，后端接口对齐了吗？建议先确认 API contract。")
		}
		if hasBackend && !hasFrontend {
			questions = append(questions,
				"💡 只改了后端，前端对接准备好了吗？")
		}
	}

	// Check if stuck (no todo completions recently)
	hasTodoCompletion := false
	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		if e.EventType == "todo" && e.TodoDone > 0 {
			hasTodoCompletion = true
			break
		}
	}
	if !hasTodoCompletion && len(events) > 0 {
		questions = append(questions,
			"💡 最近 10 分钟没有完成待办——是不是卡住了？换个角度聊聊？")
	}

	// Check new dependencies
	for _, e := range events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
			base := filepath.Base(e.Detail)
			if base == "go.mod" || base == "package.json" || base == "Cargo.toml" {
				questions = append(questions,
					"💡 新增了依赖。确认过选型吗？有无更轻量的替代？")
				break
			}
		}
	}

	// Always show scope goals as reference
	if scope != nil && len(scope.Goals) > 0 {
		questions = append([]string{"📋 当前目标：\n   " + strings.Join(scope.Goals, "\n   ")}, questions...)
	}

	return questions
}

func isInScope(filePath string, scopeFiles []string) bool {
	for _, pattern := range scopeFiles {
		// Simple glob: prefix or suffix match
		if strings.HasPrefix(pattern, "*.") {
			if strings.HasSuffix(filePath, strings.TrimPrefix(pattern, "*")) {
				return true
			}
		} else if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(filePath, dir) {
				return true
			}
		} else if strings.HasSuffix(pattern, "/**") {
			dir := strings.TrimSuffix(pattern, "/**")
			if strings.HasPrefix(filePath, dir) {
				return true
			}
		} else if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}
	return false
}
