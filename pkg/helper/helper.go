package helper

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/pkg/fmt"
)

var reMainFunc = regexp.MustCompile(`func\s+main\s*\(`)

func GetProjectName(dir string) string {
	modFile, err := os.Open(dir + "/go.mod")
	if err != nil {
		fmt.Error("go.mod does not exist error: %s", err)
		return ""
	}
	defer func(modFile *os.File) {
		_ = modFile.Close()
	}(modFile)

	var moduleName string
	_, err = fmt.Fscanf(modFile, "module %s", &moduleName)
	if err != nil {
		fmt.Error("read go mod error: %s", err)
		return ""
	}
	return moduleName
}

func SplitArgs(cmd *cobra.Command, args []string) (cmdArgs, programArgs []string) {
	dashAt := cmd.ArgsLenAtDash()
	if dashAt >= 0 {
		return args[:dashAt], args[dashAt:]
	}
	return args, []string{}
}

func FindMain(base, excludeDir string) (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(wd, "/") {
		wd += "/"
	}
	excludeDirArr := strings.Split(excludeDir, ",")
	for i := range excludeDirArr {
		excludeDirArr[i] = strings.TrimSpace(excludeDirArr[i])
	}
	cmdPath := make(map[string]string)
	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && isExcluded(path, excludeDirArr) {
			return filepath.SkipDir
		}
		if !info.IsDir() && filepath.Ext(path) == ".go" {
			// 只检查 cmd/ 目录下的文件，避免读取无关文件
			if !strings.Contains(path, "cmd"+string(filepath.Separator)) &&
				!strings.Contains(path, "cmd/") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !strings.Contains(string(content), "package main") {
				return nil
			}
			if reMainFunc.Match(content) {
				absPath, absErr := filepath.Abs(path)
				if absErr != nil {
					return absErr
				}
				d, _ := filepath.Split(absPath)
				cmdPath[strings.TrimPrefix(absPath, wd)] = d
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cmdPath, nil
}

// isExcluded 判断 path 是否位于任一排除目录下（或本身就是排除目录）。
// 通过逐段匹配路径分隔符，避免 HasPrefix 因 ./ 前缀或 .git 误伤 .gitignore 等问题。
func isExcluded(path string, excludeDirs []string) bool {
	clean := filepath.Clean(path)
	for _, dir := range excludeDirs {
		if dir == "" {
			continue
		}
		if clean == dir || strings.HasPrefix(clean, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
