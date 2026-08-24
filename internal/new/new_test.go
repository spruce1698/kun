package new

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ZipDir 将 srcDir 目录下的所有文件打包至 dstZip 文件中
func ZipDir(srcDir, dstZip string) error {
	zipFile, err := os.Create(dstZip)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		// 统一使用 POSIX 风格路径分隔符
		relPath = filepath.ToSlash(relPath)

		// 忽略 .git 目录及临时文件
		if strings.HasPrefix(relPath, ".git/") || relPath == ".git" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = relPath
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func TestPackAndExtractTemplates(t *testing.T) {
	// 获取 tpl 根路径
	tplDir := filepath.Join("..", "..", "tpl")
	basicDir := filepath.Join(tplDir, "basic")
	advancedDir := filepath.Join(tplDir, "advanced")

	basicZip := filepath.Join(tplDir, "basic.zip")
	advancedZip := filepath.Join(tplDir, "advanced.zip")

	// 1. 打包 basic
	if err := ZipDir(basicDir, basicZip); err != nil {
		t.Fatalf("failed to pack basic.zip: %v", err)
	}

	// 2. 打包 advanced
	if err := ZipDir(advancedDir, advancedZip); err != nil {
		t.Fatalf("failed to pack advanced.zip: %v", err)
	}

	// 3. 测试解压到临时目录
	tmpDir, err := os.MkdirTemp("", "kun-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	basicOut := filepath.Join(tmpDir, "test-basic")
	if err := handlerZip(basicOut, "basic"); err != nil {
		t.Fatalf("handlerZip basic failed: %v", err)
	}

	// 校验解压文件是否存在
	if _, err := os.Stat(filepath.Join(basicOut, "go.mod")); err != nil {
		t.Fatalf("basic go.mod not found after extraction: %v", err)
	}

	advOut := filepath.Join(tmpDir, "test-advanced")
	if err := handlerZip(advOut, "advanced"); err != nil {
		t.Fatalf("handlerZip advanced failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(advOut, "go.mod")); err != nil {
		t.Fatalf("advanced go.mod not found after extraction: %v", err)
	}
}
