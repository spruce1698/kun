package new

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/config"
	"github.com/spruce1698/kun/pkg/helper"
	"github.com/spruce1698/kun/pkg/output"
	"github.com/spruce1698/kun/tpl"
)

type Project struct {
	ProjectName string `survey:"name"`
}

var CmdNew = &cobra.Command{
	Use:     "new",
	Example: "kun new demo",
	Short:   "create a new project.",
	Long:    `create a new project with kun layout.`,
	Run:     run,
}
var (
	repoURL string
)

func init() {
	CmdNew.Flags().StringVarP(&repoURL, "repo-url", "g", repoURL, "layout repo")
}

func NewProject() *Project {
	return &Project{}
}

func run(_ *cobra.Command, args []string) {
	p := NewProject()
	switch len(args) {
	case 0:
		err := survey.AskOne(&survey.Input{
			Message: "What is your project name?",
			Help:    "project name.",
			Suggest: nil,
		}, &p.ProjectName, survey.WithValidator(survey.Required))
		if err != nil {
			return
		}
	case 1:
		p.ProjectName = args[0]
	default:
		output.Error("accepts %d arg(s), received %d", 1, len(args))
		return
	}

	// clone repo
	yes, err := p.cloneTemplate()
	if err != nil || !yes {
		return
	}

	err = p.replacePackageName()
	if err != nil {
		return
	}
	err = p.modTidy()
	if err != nil {
		return
	}
	p.rmGit()
	if err := p.installWire(); err != nil {
		output.Error("project created but wire install failed: %s", err)
		return
	}
	output.Success("Project [ %s ] created successfully!", p.ProjectName)
	output.Success("Done. Now run:")
	output.Success("› cd %s ", p.ProjectName)
	output.Success("› kun run \n")
}

func (p *Project) cloneTemplate() (bool, error) {

	stat, _ := os.Stat(p.ProjectName)
	if stat != nil {
		var overwrite = false

		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Folder %s already exists, do you want to overwrite it?", p.ProjectName),
			Help:    "Remove old project and create new project.",
		}
		err := survey.AskOne(prompt, &overwrite)
		if err != nil {
			return false, err
		}
		if !overwrite {
			return false, nil
		}
		err = os.RemoveAll(p.ProjectName)
		if err != nil {
			output.Error("remove old project error: %s", err)
			return false, err
		}
	}

	if repoURL == "" {
		layout := ""
		prompt := &survey.Select{
			Message: "Please select a layout:",
			Options: []string{
				"Advanced",
				"Basic",
			},
			Description: func(value string, index int) string {
				if index == 1 {
					return "A basic project structure"
				}
				return "It has rich functions such as db, jwt, cron, migration, test, etc"
			},
		}
		err := survey.AskOne(prompt, &layout)
		if err != nil {
			return false, err
		}
		err = os.RemoveAll(p.ProjectName)
		if err != nil {
			output.Error("remove old project error: %s", err)
			return false, err
		}

		templateName := "basic"
		if layout != "Basic" {
			templateName = "advanced"
		}

		output.Success("Generate code from template: %s", templateName)

		err = handlerZip(p.ProjectName, templateName)
		if err != nil {
			output.Error("Generate code from template: %s, error: %s", templateName, err)
			return false, err
		}

	} else { // clone from repoURL
		output.Success("git clone %s", repoURL)
		cmd := exec.Command("git", "clone", repoURL, p.ProjectName)
		_, err := cmd.CombinedOutput()
		if err != nil {
			output.Error("git clone %s error: %s", repoURL, err)
			return false, err
		}
	}
	return true, nil
}

func (p *Project) replacePackageName() error {
	packageName, err := helper.GetProjectName(p.ProjectName)
	if err != nil {
		return err
	}

	err = p.replaceFiles(packageName)
	if err != nil {
		return err
	}

	cmd := exec.Command("go", "mod", "edit", "-module", p.ProjectName)
	cmd.Dir = p.ProjectName
	_, err = cmd.CombinedOutput()
	if err != nil {
		output.Error("go mod edit error: %s", err)
		return err
	}
	return nil
}
func (p *Project) modTidy() error {
	output.Success("go mod tidy")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = p.ProjectName
	if err := cmd.Run(); err != nil {
		output.Error("go mod tidy error: %s", err)
		return err
	}
	return nil
}
func (p *Project) rmGit() {
	_ = os.RemoveAll(p.ProjectName + "/.git")
}
func (p *Project) installWire() error {
	output.Success("go install %s", config.WireUrl)
	cmd := exec.Command("go", "install", config.WireUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install %s: %w", config.WireUrl, err)
	}
	return nil
}

func (p *Project) replaceFiles(packageName string) error {
	// 只替换 import path 前缀("packageName/"),避免全文替换误伤注释/字符串里的同名词。
	// 例如 packageName="advanced" 时,"advanced/pkg/xlog" -> "myproj/pkg/xlog",
	// 但不会改动注释里的 "advanced usage" 或字符串 "basic auth"。
	oldImport := "\"" + packageName + "/"
	newImport := "\"" + p.ProjectName + "/"
	err := filepath.Walk(p.ProjectName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		newData := bytes.ReplaceAll(data, []byte(oldImport), []byte(newImport))
		if err := os.WriteFile(path, newData, 0644); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		output.Error("walk file error: %s", err)
		return err
	}
	return nil
}

func handlerZip(projectName, templateName string) error {
	// 创建项目目录
	mkDirErr := os.MkdirAll(projectName, os.ModePerm)
	if mkDirErr != nil {
		return mkDirErr
	}
	tempFile, readErr := tpl.NewTplZipFS.ReadFile(templateName + ".zip")
	if readErr != nil {
		return readErr
	}

	zipReader, zipReaderErr := zip.NewReader(bytes.NewReader(tempFile), int64(len(tempFile)))
	if zipReaderErr != nil {
		return zipReaderErr
	}

	// 遍历 zip 包里的文件
	baseAbs, err := filepath.Abs(projectName)
	if err != nil {
		return err
	}
	for _, file := range zipReader.File {
		// 防 zip slip：拒绝任何逃逸出目标目录的路径
		name := filepath.Clean(filepath.FromSlash(file.Name))
		if strings.HasPrefix(name, ".."+string(filepath.Separator)) || filepath.IsAbs(name) {
			return fmt.Errorf("zip entry %q has unsafe path", file.Name)
		}
		path := filepath.Join(projectName, name)
		if !strings.HasPrefix(filepath.Clean(path), baseAbs+string(filepath.Separator)) && filepath.Clean(path) != baseAbs {
			return fmt.Errorf("zip entry %q escapes target directory", file.Name)
		}
		fileMode := file.Mode()
		// 如果是目录，就创建目录
		if file.FileInfo().IsDir() {
			if tempMkDirErr := os.MkdirAll(path, fileMode); tempMkDirErr != nil {
				return tempMkDirErr
			}
			// 因为是目录，跳过当前循环，因为后面都是文件的处理
			continue
		}

		// 获取到 Reader
		fr, frErr := file.Open()
		if frErr != nil {
			return frErr
		}

		// 创建要写出的文件对应的 Write
		fw, fwErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fileMode)
		if fwErr != nil {
			_ = fr.Close()
			return fwErr
		}

		_, copyErr := io.Copy(fw, fr)
		if copyErr != nil {
			_ = fw.Close()
			_ = fr.Close()
			return copyErr
		}
		_ = fw.Close()
		_ = fr.Close()
	}

	return nil
}

// func handlerFiles(projectName, templateName string) error {
// 	// 创建项目目录
// 	mkDirErr := os.MkdirAll(projectName, os.ModeExclusive)
// 	if mkDirErr != nil {
// 		return mkDirErr
// 	}
// 	files, _ := fs.ReadDir(tpl.NewTplDirFS, templateName)
// 	for _, file := range files {
// 		fileName := file.Name()
// 		fileMode := file.Type()
// 		path := filepath.Join(projectName, fileName)
// 		if fileName == "go.mod.tpl" {
// 			path = filepath.Join(projectName, "go.mod")
// 		}
// 		// 如果是目录，就创建目录
// 		if file.IsDir() {
// 			if mkdirErr := os.MkdirAll(path, fileMode); mkdirErr != nil {
// 				return mkdirErr
// 			}
// 			// 因为是目录，跳过当前循环，因为后面都是文件的处理
// 			continue
// 		}
//
// 		// 获取到 Reader
// 		fr, frErr := fs.ReadFile(tpl.NewTplDirFS, templateName+"/"+fileName)
// 		if frErr != nil {
// 			return frErr
// 		}
//
// 		// 创建要写出的文件对应的 Write
// 		fw, fwErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fileMode)
// 		if fwErr != nil {
// 			_ = fw.Close()
// 			return fwErr
// 		}
// 		_, writeErr := fw.Write(fr)
// 		if writeErr != nil {
// 			_ = fw.Close()
// 			return writeErr
// 		}
// 	}
// 	return nil
// }
