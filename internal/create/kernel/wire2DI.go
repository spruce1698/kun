package kernel

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Wire2DIFile 在 filePath 指向的位置扫描 DI 文件并写入注入内容。
// filePath 既可以是 DI 文件所在目录,也可以是 DI 文件本身(此时取其所在目录),
// 还可以是生成代码所在目录(此时向上逐级查找含 marker 的 DI 文件)。
func Wire2DIFile(filePath string, contentMap map[string]string) error {
	// 确定起始扫描目录:filePath 为目录则直接用,为文件则取其所在目录。
	dirName := filePath
	if info, err := os.Stat(dirName); err == nil && !info.IsDir() {
		dirName = filepath.Dir(dirName)
	}

	// 向上逐级查找含任一 marker 的 DI 文件,最多上探 4 级,覆盖
	// db/cache(生成目录的父级)与 hdl/svc/router(生成目录本身)两种 DI 文件布局。
	diFiles := findDIFiles(dirName, contentMap, 4)
	if len(diFiles) == 0 {
		return fmt.Errorf("the DI file does not exist near %s", filePath)
	}

	for _, diFile := range diFiles {
		for markerLine, appendContent := range contentMap {
			if err := wireProcess(diFile, markerLine, appendContent); err != nil {
				return fmt.Errorf("processing files %s Failed: %w", diFile, err)
			}
		}
	}
	return nil
}

// findDIFiles 从 dir 开始向上逐级(最多 maxUp 级)扫描,返回所有包含 contentMap 中任一 marker 的文件。
func findDIFiles(dir string, contentMap map[string]string, maxUp int) []string {
	var diFiles []string
	for i := 0; i <= maxUp; i++ {
		fileInfos, err := os.ReadDir(dir)
		if err != nil {
			break
		}
		for _, fileInfo := range fileInfos {
			if fileInfo.IsDir() {
				continue
			}
			p := filepath.Join(dir, fileInfo.Name())
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
		if len(diFiles) > 0 {
			return diFiles
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // 已到文件系统根
		}
		dir = parent
	}
	return diFiles
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
