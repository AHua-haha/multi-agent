package service

import (
	"bufio"
	"errors"
	"fmt"
	"multi-agent/shared"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"mvdan.cc/sh/v3/syntax"
)

func CreateTempFIle(dir string, pattern string) (string, error) {
	// "" uses the default system temp directory (usually /tmp on Linux)
	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

func ExtractAllBinaries(shellCmd string) ([]string, error) {
	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(shellCmd), "")
	if err != nil {
		return nil, fmt.Errorf("error when parsing shell command: %v", err)
	}

	var binaries []string
	syntax.Walk(file, func(node syntax.Node) bool {
		switch x := node.(type) {
		case *syntax.CallExpr:
			if len(x.Args) > 0 {
				binName := x.Args[0].Lit()
				if binName != "" {
					binaries = append(binaries, binName)
				}
			}
		}
		return true // Continue walking the rest of the tree
	})

	return binaries, nil
}

func ViewFile(file string, lines [][]int) (string, error) {

	validLines := [][]int{}
	for _, line := range lines {
		if line[0] <= line[1] {
			validLines = append(validLines, line)
		}
	}
	sort.Slice(validLines, func(i, j int) bool {
		return validLines[i][0] < validLines[j][0]
	})
	var builder strings.Builder
	linecnt, err := shared.CountLines(file)
	if err != nil {
		return "", err
	}
	builder.WriteString(fmt.Sprintf("File %s total lines: %d\n", file, linecnt))
	mergeLines := [][]int{}
	for idx := 0; idx < len(validLines); {
		start := validLines[idx][0]
		end := validLines[idx][1]
		for idx < len(validLines) && validLines[idx][0] <= end+10 {
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
	builder.WriteString("```text\n")
	idx := 0
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		builder.WriteString("```\n")
		return builder.String(), nil
	}
	currentLine := 1

RangeIter:
	for idx < len(mergeLines) {
		start := mergeLines[idx][0]
		end := mergeLines[idx][1]
		if currentLine < start {
			builder.WriteString(fmt.Sprintf("# ... [lines %d-%d omitted] ...\n", currentLine, start-1))
		}
		for currentLine != start {
			if !scanner.Scan() {
				break RangeIter
			}
			currentLine++
		}
		for currentLine <= end {
			builder.WriteString(fmt.Sprintf("%4d | %s\n", currentLine, scanner.Text()))
			if !scanner.Scan() {
				break RangeIter
			}
			currentLine++
		}
		idx++
	}
	builder.WriteString("```\n")
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

var readOnlyWhitelist = map[string]bool{
	"ls": true, "cat": true, "grep": true, "pwd": true, "find": true,
	"head": true, "tail": true, "wc": true, "du": true, "df": true,
	"ps": true, "whoami": true, "file": true, "stat": true,
	"cd": true, "rg": true,
}

type RunMode int

const (
	Exit RunMode = iota
	DirectRun
	DiffRun
)

type BashTool struct {
	tempIndexFile string
	gitRepoPath   string
}

type BashRes struct {
	ExitCode      int      `json:"exit_code"`
	Output        string   `json:"output"`                   // Combined Stdout and Stderr
	ModifiedFiles []string `json:"modified_files,omitempty"` // Files that existed and changed
	CreatedFiles  []string `json:"created_files,omitempty"`  // Brand new files
	DeletedFiles  []string `json:"deleted_files,omitempty"`
}

func (tool *BashTool) AddRepo(path string) error {
	tmpFile, err := CreateTempFIle("", "agent_temp_index_")
	if err != nil {
		log.Error().Err(err).Msg("Create temp file failed")
		return err
	}

	tool.gitRepoPath = path
	tool.tempIndexFile = tmpFile

	initCmd := exec.Command("git", "read-tree", "HEAD")
	initCmd.Dir = tool.gitRepoPath
	initCmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tool.tempIndexFile)
	err = initCmd.Run()
	if err != nil {
		log.Error().Err(err).Any("cmd", initCmd.String()).Msg("init cmd run failed")
		return err
	}
	log.Info().Any("git repo", tool.gitRepoPath).Any("tmp index", tool.tempIndexFile).Msg("init bash tool")
	return nil
}
func (tool *BashTool) Run(cmd string, dir string) (*BashRes, error) {
	runMode, err := tool.chooseRunMode(cmd, dir)
	if err != nil {
		log.Error().Err(err).Any("cmd", cmd).Msg("choose run mode for shell cmd failed")
		return nil, err
	}
	var res *BashRes
	switch runMode {
	case DirectRun:
		res, err = tool.DirectRun(cmd, dir)
	case DiffRun:
		res, err = tool.DiffRun(cmd, dir)
	}
	if err != nil {
		log.Error().Err(err).Any("cmd", cmd).Msg("Executing shell cmd failed")
		return nil, err
	}
	log.Info().Any("cmd", cmd).Any("Run Mode", runMode).Msg("Executing shell cmd success")
	return res, nil
}
func (tool *BashTool) chooseRunMode(cmd string, dir string) (RunMode, error) {
	binarys, err := ExtractAllBinaries(cmd)
	if err != nil {
		return Exit, err
	}
	for _, name := range binarys {
		if _, exist := readOnlyWhitelist[name]; !exist {
			return DiffRun, nil
		}
	}
	return DirectRun, nil
}
func (tool *BashTool) DirectRun(cmd string, dir string) (*BashRes, error) {
	runCmd := exec.Command("bash", "-c", cmd)
	if dir != "" {
		runCmd.Dir = dir
	}
	output, err := runCmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return nil, err // System error (e.g., bash not found)
		}
	}
	bashResult := &BashRes{
		ExitCode: exitCode,
		Output:   string(output),
	}
	return bashResult, nil
}
func (tool *BashTool) DiffRun(cmd string, dir string) (*BashRes, error) {
	err := tool.syncRepo()
	if err != nil {
		return nil, err
	}
	runCmd := exec.Command("bash", "-c", cmd)
	if dir != "" {
		runCmd.Dir = dir
	}
	output, err := runCmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return nil, err // System error (e.g., bash not found)
		}
	}
	bashResult := &BashRes{
		ExitCode: exitCode,
		Output:   string(output),
	}
	tool.diffOutput(bashResult)

	return bashResult, nil
}
func (tool *BashTool) syncRepo() error {
	addCmd := exec.Command("git", "add", "--all")
	addCmd.Dir = tool.gitRepoPath
	addCmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tool.tempIndexFile)
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add files: %v", err)
	}

	refreshCmd := exec.Command("git", "update-index", "--refresh", "--really-refresh")
	refreshCmd.Dir = tool.gitRepoPath
	refreshCmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tool.tempIndexFile)
	_ = refreshCmd.Run()
	return nil
}

func (tool *BashTool) diffOutput(res *BashRes) {
	cmd := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all")
	cmd.Dir = tool.gitRepoPath
	cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tool.tempIndexFile)

	out, _ := cmd.Output()
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		status := line[:2]
		filename := line[3:]

		switch status {
		case " M":
			res.ModifiedFiles = append(res.ModifiedFiles, filename)
		case " D":
			res.DeletedFiles = append(res.ModifiedFiles, filename)
		case "??":
			res.CreatedFiles = append(res.ModifiedFiles, filename)
		}
	}
}
