// Package executor provides command and file execution capabilities.
package executor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

const (
	maxFileSize = 50 * 1024 // 50KB
	maxLines    = 2000
)

// FileExecutor handles file read/write/edit operations with path security.
type FileExecutor struct {
	rootPath string
}

// NewFileExecutor creates a FileExecutor rooted at the given path.
func NewFileExecutor(rootPath string) *FileExecutor {
	return &FileExecutor{rootPath: rootPath}
}

// validatePath resolves and validates a path, preventing directory traversal.
func (f *FileExecutor) validatePath(inputPath string) (string, bool) {
	resolved := filepath.Join(f.rootPath, inputPath)
	resolved = filepath.Clean(resolved)

	rel, err := filepath.Rel(f.rootPath, resolved)
	if err != nil {
		return "", false
	}
	if strings.HasPrefix(rel, "..") {
		return "", false
	}
	return resolved, true
}

// Read reads a file with offset/limit support and truncation.
func (f *FileExecutor) Read(payload protocol.ReadPayload) protocol.FileResult {
	validPath, ok := f.validatePath(payload.Path)
	if !ok {
		return protocol.FileResult{Error: "Invalid path: cannot access outside root directory"}
	}

	data, err := os.ReadFile(validPath)
	if err != nil {
		if os.IsNotExist(err) {
			return protocol.FileResult{Error: "File not found"}
		}
		return protocol.FileResult{Error: "Failed to read file: " + err.Error()}
	}

	lines := strings.Split(string(data), "\n")

	startLine := 0
	if payload.Offset != nil {
		startLine = *payload.Offset - 1
	}
	if startLine < 0 {
		startLine = 0
	}

	limit := maxLines
	if payload.Limit != nil && *payload.Limit < maxLines {
		limit = *payload.Limit
	}

	if startLine >= len(lines) {
		return protocol.FileResult{Content: "", LinesRead: 0}
	}

	endLine := startLine + limit
	if endLine > len(lines) {
		endLine = len(lines)
	}

	selected := lines[startLine:endLine]
	result := strings.Join(selected, "\n")
	linesRead := len(selected)
	truncated := endLine < len(lines)

	if len(result) > maxFileSize {
		result = result[:maxFileSize]
		truncated = true
	}

	return protocol.FileResult{
		Content:   result,
		LinesRead: linesRead,
		Truncated: truncated,
	}
}

// Write writes content to a file, creating parent directories as needed.
func (f *FileExecutor) Write(payload protocol.WritePayload) protocol.FileResult {
	validPath, ok := f.validatePath(payload.Path)
	if !ok {
		return protocol.FileResult{Error: "Invalid path: cannot access outside root directory"}
	}

	parentDir := filepath.Dir(validPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return protocol.FileResult{Error: "Failed to create directory: " + err.Error()}
	}

	if err := os.WriteFile(validPath, []byte(payload.Content), 0644); err != nil {
		return protocol.FileResult{Error: "Failed to write file: " + err.Error()}
	}

	return protocol.FileResult{}
}

// Edit replaces exact text in a file.
func (f *FileExecutor) Edit(payload protocol.EditPayload) protocol.FileResult {
	validPath, ok := f.validatePath(payload.Path)
	if !ok {
		return protocol.FileResult{Error: "Invalid path: cannot access outside root directory"}
	}

	data, err := os.ReadFile(validPath)
	if err != nil {
		if os.IsNotExist(err) {
			return protocol.FileResult{Error: "File not found"}
		}
		return protocol.FileResult{Error: "Failed to read file: " + err.Error()}
	}

	content := string(data)
	idx := strings.Index(content, payload.OldText)
	if idx == -1 {
		return protocol.FileResult{Error: "Old text not found in file"}
	}

	newContent := content[:idx] + payload.NewText + content[idx+len(payload.OldText):]
	if err := os.WriteFile(validPath, []byte(newContent), 0644); err != nil {
		return protocol.FileResult{Error: "Failed to write file: " + err.Error()}
	}

	return protocol.FileResult{}
}
