package rules

import (
	"path/filepath"
	"strings"

	"github.com/francis/vibeship/internal/config"
	"github.com/francis/vibeship/internal/store"
)

// RecommendSkill returns a skill name to suggest based on recent activity.
// Events are already filtered to the recent time window by the caller.
func RecommendSkill(events []store.TranscriptEvent) string {
	hasEdit := false
	hasBashError := false
	hasGitCommit := false
	lastSkill := ""

	for _, e := range events {
		switch e.EventType {
		case "tool_call":
			if e.Name == "Write" || e.Name == "Edit" {
				hasEdit = true
			}
			if e.Name == "Bash" && strings.Contains(strings.ToLower(e.Detail), "error") {
				hasBashError = true
			}
			if e.Name == "Bash" && strings.Contains(e.Detail, "git commit") {
				hasGitCommit = true
			}
		case "skill":
			lastSkill = e.Name
		}
	}

	if hasEdit {
		return "code-review: 刚改完代码，review 一下？"
	}
	if hasBashError {
		return "systematic-debugging: 检测到错误，调试一下？"
	}
	if hasGitCommit {
		return "code-review: 刚提交了代码"
	}
	if lastSkill != "" {
		return lastSkill + ": 继续用这个 skill？"
	}

	// Default: suggest based on common vibe coding patterns
	return "brainstorming: 聊聊思路？"
}

// GenerateCheckQuestions produces scope/thinking questions based on recent activity.
func GenerateCheckQuestions(scope *config.Scope, events []store.TranscriptEvent) []string {
	var questions []string

	if scope == nil {
		questions = append(questions, "💡 建议创建 SCOPE.md 锁定范围")
		questions = append(questions, "💡 现在在做什么功能？需要聊聊思路吗？")
		return questions
	}

	// Check if recent file edits are within scope
	var outOfScopeFiles []string
	for _, e := range events {
		if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
			if e.Detail == "" {
				continue
			}
			if !isInScope(e.Detail, scope.Files) {
				outOfScopeFiles = append(outOfScopeFiles, e.Detail)
			}
		}
	}

	if len(outOfScopeFiles) > 0 {
		questions = append(questions,
			"⚠️ 改动了 SCOPE.md 之外的文件：\n   "+strings.Join(outOfScopeFiles, ", ")+"\n   确定继续？")
	}

	// Check if writing code without schema changes
	hasWrite := false
	hasSchema := false
	for _, e := range events {
		if e.EventType == "tool_call" && (e.Name == "Write" || e.Name == "Edit") {
			hasWrite = true
			if strings.Contains(e.Detail, "migration") ||
				strings.Contains(e.Detail, "schema") ||
				strings.Contains(e.Detail, ".sql") {
				hasSchema = true
			}
		}
	}
	if hasWrite && !hasSchema && len(scope.DevelopOrder) > 0 {
		questions = append(questions,
			"💡 开发顺序：\n   "+strings.Join(scope.DevelopOrder, " → "))
	}

	// Check frontend/backend coordination
	hasFrontend := false
	hasBackend := false
	for _, e := range events {
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
		questions = append(questions, "💡 只改了前端，后端接口对齐了吗？")
	}
	if hasBackend && !hasFrontend {
		questions = append(questions, "💡 只改了后端，前端对接好了吗？")
	}

	// Show scope goals as reference
	if len(scope.Goals) > 0 {
		questions = append([]string{"📋 " + strings.Join(scope.Goals, " · ")}, questions...)
	}

	if len(questions) == 0 {
		questions = append(questions, "💡 一切正常，保持节奏 🚀")
	}

	return questions
}

func isInScope(filePath string, scopeFiles []string) bool {
	for _, pattern := range scopeFiles {
		if strings.HasPrefix(pattern, "*.") {
			if strings.HasSuffix(filePath, strings.TrimPrefix(pattern, "*")) {
				return true
			}
		} else if strings.HasSuffix(pattern, "/**") {
			dir := strings.TrimSuffix(pattern, "/**") + "/"
			if strings.HasPrefix(filePath, dir) {
				return true
			}
		} else if strings.HasSuffix(pattern, "/*") {
			dir := strings.TrimSuffix(pattern, "/*") + "/"
			if strings.HasPrefix(filePath, dir) {
				rest := strings.TrimPrefix(filePath, dir)
				if !strings.Contains(rest, "/") {
					return true
				}
			}
		} else if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}
	return len(scopeFiles) == 0 // if no scope defined, everything is in scope
}
