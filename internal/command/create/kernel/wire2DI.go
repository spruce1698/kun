package kernel

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/spruce1698/kun/pkg/fmt"
)

// 写入都DI文件,
func Wire2DIFile(filePath string, contentMap map[string]string) error {
	// 构建目录路径
	dirName := filepath.Join(filePath, "../")
	// 读取目录内容
	fileInfos, err := os.ReadDir(dirName)
	if err != nil {
		return fmt.Errorf("failed to read the directory: %w", err)
	}

	// 找出确实包含任一 marker 的 DI 文件，避免对无关文件做无意义的改写
	diFiles := make([]string, 0, len(fileInfos))
	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}
		p := filepath.Join(dirName, fileInfo.Name())
		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for markerLine := range contentMap {
			if bytes.Contains(content, []byte(markerLine)) {
				diFiles = append(diFiles, p)
				break
			}
		}
	}

	if len(diFiles) == 0 {
		return fmt.Errorf("the DI file does not exist")
	}

	for _, diFile := range diFiles {
		for markerLine, appendContent := range contentMap {
			if err = wireProcess(diFile, markerLine, appendContent); err != nil {
				return fmt.Errorf("processing files %s Failed: %w", diFile, err)
			}
		}
	}
	return nil
}

// 写入内容到文件
func wireProcess(filePath, markerLine, appendContent string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 检查是否已包含要插入的内容
	for _, v := range strings.Split(appendContent, "\n") {
		if bytes.Contains(content, []byte(strings.Trim(strings.Trim(v, " "), "\t"))) {
			return nil
		}
	}

	// 处理文件内容
	lines := strings.Split(string(content), "\n")
	var newContent []string

	markerFound := false
	for _, line := range lines {
		if strings.Contains(line, markerLine) {
			newContent = append(newContent, appendContent)
			markerFound = true
		}
		newContent = append(newContent, line)
	}

	// 如果没有找到标记行，不进行修改
	if !markerFound {
		return nil
	}

	// 写回文件
	return os.WriteFile(filePath, []byte(strings.Join(newContent, "\n")), 0644)
}
