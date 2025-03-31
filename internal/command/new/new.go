package new

import (
    "archive/zip"
    "bytes"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    
    "github.com/AlecAivazis/survey/v2"
    "github.com/spf13/cobra"
    "github.com/spruce1698/kun/config"
    "github.com/spruce1698/kun/pkg/fmt"
    "github.com/spruce1698/kun/pkg/helper"
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
        fmt.Error("accepts %d arg(s), received %d", 1, len(args))
        return
    }
    
    // clone repo
    yes, err := p.cloneTemplate()
    if err != nil || !yes {
        return
    }
    
    err = p.replacePackageName()
    if err != nil || !yes {
        return
    }
    
    err = p.replacePackageName()
    if err != nil || !yes {
        return
    }
    err = p.modTidy()
    if err != nil || !yes {
        return
    }
    p.rmGit()
    p.installWire()
    fmt.Success("Project [ %s ] created successfully!", p.ProjectName)
    fmt.Success("Done. Now run:")
    fmt.Success("› cd %s ", p.ProjectName)
    fmt.Success("› kun run \n")
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
            fmt.Error("remove old project error: %s", err)
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
            fmt.Error("remove old project error: %s", err)
            return false, err
        }
        
        templateName := "basic"
        if layout != "Basic" {
            templateName = "advanced"
        }
        
        fmt.Success("Generate code from template: %s", templateName)
        
        err = handlerZip(p.ProjectName, templateName)
        if err != nil {
            fmt.Error("Generate code from template: %s, error: %s", templateName, err)
            return false, err
        }
        
    } else { // clone from repoURL
        fmt.Success("git clone %s", repoURL)
        cmd := exec.Command("git", "clone", repoURL, p.ProjectName)
        _, err := cmd.CombinedOutput()
        if err != nil {
            fmt.Error("git clone %s error: %s", repoURL, err)
            return false, err
        }
    }
    return true, nil
}

func (p *Project) replacePackageName() error {
    packageName := helper.GetProjectName(p.ProjectName)
    
    err := p.replaceFiles(packageName)
    if err != nil {
        return err
    }
    
    cmd := exec.Command("go", "mod", "edit", "-module", p.ProjectName)
    cmd.Dir = p.ProjectName
    _, err = cmd.CombinedOutput()
    if err != nil {
        fmt.Error("go mod edit error: %s", err)
        return err
    }
    return nil
}
func (p *Project) modTidy() error {
    fmt.Success("go mod tidy")
    cmd := exec.Command("go", "mod", "tidy")
    cmd.Dir = p.ProjectName
    if err := cmd.Run(); err != nil {
        fmt.Error("go mod tidy error: %s", err)
        return err
    }
    return nil
}
func (p *Project) rmGit() {
    _ = os.RemoveAll(p.ProjectName + "/.git")
}
func (p *Project) installWire() {
    fmt.Success("go install %s", config.WireCmd)
    cmd := exec.Command("go", "install", config.WireCmd)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        fmt.Error("go install %s error", err)
    }
}

func (p *Project) replaceFiles(packageName string) error {
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
        newData := bytes.ReplaceAll(data, []byte(packageName), []byte(p.ProjectName))
        if err := os.WriteFile(path, newData, 0644); err != nil {
            return err
        }
        return nil
    })
    if err != nil {
        fmt.Error("walk file error: %s", err)
        return err
    }
    return nil
}

func handlerZip(projectName, templateName string) error {
    // 创建项目目录
    mkDirErr := os.MkdirAll(projectName, os.ModeExclusive)
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
    for _, file := range zipReader.File {
        path := filepath.Join(projectName, file.Name)
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
            _ = fr.Close()
            return frErr
        }
        
        // 创建要写出的文件对应的 Write
        fw, fwErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fileMode)
        if fwErr != nil {
            _ = fw.Close()
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
