package service

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func ViewFile(file string, lines [][]int) (string, error) {
	var builder strings.Builder

	validLines := [][]int{}
	for _, line := range lines {
		if line[0] <= line[1] {
			validLines = append(validLines, line)
		}
	}
	sort.Slice(validLines, func(i, j int) bool {
		return validLines[i][0] < validLines[j][0]
	})
	mergeLines := [][]int{}
	for idx := 0; idx < len(validLines); {
		start := validLines[idx][0]
		end := validLines[idx][1]
		for validLines[idx][0] <= end {
			end = max(end, validLines[idx][1])
			idx++
		}
		mergeLines = append(mergeLines, []int{start, end})
	}
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	idx := 0
	currentLine := 1

	for scanner.Scan() {
		if currentLine >= mergeLines[idx][0] && currentLine <= mergeLines[idx][1] {
			builder.WriteString(fmt.Sprintf("%d|%s\n", currentLine, scanner.Text()))
		}
		if currentLine == mergeLines[idx][1] {
			idx++
			if idx >= len(mergeLines) {
				break
			}
		}
		currentLine++
	}
	return builder.String(), nil
}

func EditFile(file string, unifiedDiff string) error {
	// 1. Check if the diff targets ONLY the specified file
	// Regex matches '+++ b/filename' or '+++ filename'
	fileHeaderRegex := regexp.MustCompile(`(?m)^\+\+\+ (?:b/)?([^\s\n]+)`)
	matches := fileHeaderRegex.FindAllStringSubmatch(unifiedDiff, -1)

	if len(matches) == 0 {
		return fmt.Errorf("no file headers (+++) found in diff")
	}

	for _, match := range matches {
		detectedFile := match[1]
		if detectedFile != file {
			return fmt.Errorf("security violation: diff attempts to modify %s, but only %s is allowed", detectedFile, file)
		}
	}

	// 2. Validate Hunk Order
	// Extract all hunk headers: @@ -oldStart,len +newStart,len @@
	hunkHeaderRegex := regexp.MustCompile(`@@ -(\d+),\d+ \+\d+,\d+ @@`)
	hunkMatches := hunkHeaderRegex.FindAllStringSubmatch(unifiedDiff, -1)

	lastLine := -1
	for _, match := range hunkMatches {
		currentLine, _ := strconv.Atoi(match[1])
		if currentLine < lastLine {
			return fmt.Errorf("malformed diff: hunk at line %d appears after hunk at line %d (must be in ascending order)", currentLine, lastLine)
		}
		lastLine = currentLine
	}

	// 3. Dry Run using the system patch command
	// This checks if the content/context actually matches the file on disk
	cmd := exec.Command("patch", "--dry-run", "--force", file)
	cmd.Stdin = strings.NewReader(unifiedDiff)

	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dry run failed (context mismatch): %w", err)
	}

	// 4. Actual Application
	applyCmd := exec.Command("patch", "--force", file)
	applyCmd.Stdin = strings.NewReader(unifiedDiff)
	_, err = applyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("final application failed: %w", err)
	}

	return nil
}

func CreateFile(path string, content string) error {
	// 1. Clean the path to prevent directory traversal attacks (e.g., ../../../etc/passwd)
	cleanPath := filepath.Clean(path)

	// 2. Check if the file already exists
	if _, err := os.Stat(cleanPath); err == nil {
		return fmt.Errorf("refusing to create file: '%s' already exists (use an edit tool to modify existing files)", cleanPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		// This handles permission errors or other system issues
		return fmt.Errorf("error checking file status: %w", err)
	}

	// 3. Ensure the parent directory exists
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// 4. Write the file
	// 0644 gives read/write to owner and read-only to others
	err := os.WriteFile(cleanPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
