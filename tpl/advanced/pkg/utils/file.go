package utils

import (
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"
)

// 检查目录是否存在
func CheckFileIsExist(fileName string) bool {
	var exist = true
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// 创建目录
func BuildDir(absDir string) error {
	return os.MkdirAll(path.Dir(absDir), os.ModePerm) // 生成多级目录
}

// 删除文件或文件夹
func DeleteFile(absDir string) error {
	return os.RemoveAll(absDir)
}

// 获取目录所有文件夹
func GetPathDirs(absDir string) (re []string) {
	if CheckFileIsExist(absDir) {
		files, _ := os.ReadDir(absDir)
		for _, f := range files {
			if f.IsDir() {
				re = append(re, f.Name())
			}
		}
	}
	return
}

// 获取目录所有文件
func GetPathFiles(absDir string) (re []string) {
	if CheckFileIsExist(absDir) {
		files, _ := os.ReadDir(absDir)
		for _, f := range files {
			if !f.IsDir() {
				re = append(re, f.Name())
			}
		}
	}
	return
}

// 获取程序运行目录
func GetModelPath() string {
	dir, _ := os.Getwd()
	return dir
}

// 获取exe所在目录
func GetCurrentDirectory() string {
	dir, _ := os.Executable()
	exPath := filepath.Dir(dir)
	// fmt.Println(exPath)

	return exPath
}

// 写入文件
func SaveToFile(fileName string, src []string, isClear bool) bool {
	return WriteFile(fileName, src, isClear)
}

// 写入文件
func WriteFile(fileName string, src []string, isClear bool) bool {
	BuildDir(fileName)
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if !isClear {
		flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
	}
	f, err := os.OpenFile(fileName, flag, 0666)
	if err != nil {
		return false
	}
	defer f.Close()

	for _, v := range src {
		if _, err := f.WriteString(v); err != nil {
			return false
		}
		if _, err := f.WriteString("\r\n"); err != nil {
			return false
		}
	}

	return true
}

// 读取文件
func ReadFile(fileName string) (src []string) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0666)
	if err != nil {
		return []string{}
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	for {
		line, _, rErr := rd.ReadLine()
		if rErr != nil || io.EOF == rErr {
			break
		}
		src = append(src, string(line))
	}

	return src
}
