package create

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/internal/pkg/helper"
	"github.com/spruce1698/kun/tpl"
)

type Create struct {
	ProjectName        string
	CmdType            string
	CreateType         string
	FilePath           string
	FileName           string
	FileNameTitleLower string
	FileNameFirstChar  string
	PackageName        string
	AddUPPath          string
	IsFull             bool
}

func NewCreate() *Create {
	return &Create{}
}

var CmdCreate = &cobra.Command{
	Use:     "create [type] [controller-name]",
	Short:   "Create a new ctrl/svc/repo",
	Example: "kun create ctrl user",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

	},
}
var (
	tplPath string
)

func init() {
	CmdCreateController.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateService.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateRepository.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateAll.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")
}

var CmdCreateController = &cobra.Command{
	Use:     "ctrl",
	Short:   "Create a new controller",
	Example: "kun create ctrl user",
	Args:    cobra.ExactArgs(1),
	Run:     runCreate,
}
var CmdCreateService = &cobra.Command{
	Use:     "svc",
	Short:   "Create a new service",
	Example: "kun create svc user",
	Args:    cobra.ExactArgs(1),
	Run:     runCreate,
}
var CmdCreateRepository = &cobra.Command{
	Use:     "repo",
	Short:   "Create a new repository",
	Example: "kun create repo \"name:pwd@tcp(127.0.0.1:3306)/dbname\" [t1,t2|t1|*]",
	Args:    cobra.ExactArgs(2),
	Run:     genRepo,
}
var CmdCreateAll = &cobra.Command{
	Use:     "all",
	Short:   "Create a new controller & service",
	Example: "kun create all user",
	Args:    cobra.ExactArgs(1),
	Run:     runCreate,
}

func runCreate(cmd *cobra.Command, args []string) {
	c := NewCreate()
	c.ProjectName = helper.GetProjectName(".")
	c.CmdType = cmd.Use
	c.FilePath, c.FileName = filepath.Split(args[0])
	c.FileName = strings.ReplaceAll(strings.ToUpper(string(c.FileName[0]))+c.FileName[1:], ".go", "")
	c.FileNameTitleLower = strings.ToLower(string(c.FileName[0])) + c.FileName[1:]
	c.FileNameFirstChar = string(c.FileNameTitleLower[0])

	switch c.CmdType {
	case "ctrl":
		c.CreateType = "controller"
		c.genFile()
	case "svc":
		c.CreateType = "service"
		c.genFile()
	case "all":
		c.CreateType = "controller"
		c.genFile()

		c.CreateType = "service"
		c.genFile()

	default:
		log.Fatalf("Invalid controller type: %s", c.CreateType)
	}

}

func (c *Create) genFile() {
	basePath := "internal"
	typePath := c.CreateType
	if c.CreateType == "service" {
		typePath = c.CreateType + "/svc"
	}
	filePath := c.FilePath
	if filePath == "" {
		filePath = basePath + "/" + typePath + "/"
	} else {
		c.AddUPPath = "../"
		filePath = basePath + "/" + typePath + "/" + filePath + "/"
	}
	filePath = strings.ReplaceAll(filePath, "//", "/")

	absPath, _ := filepath.Abs(filepath.Dir(filepath.Join(filePath, strings.ToLower(c.FileName)+".go")))
	absLinuxPath := strings.ReplaceAll(absPath, "\\", "/") + "/"
	dirOk := strings.Index(absLinuxPath, basePath+"/"+c.CreateType+"/")
	if dirOk == -1 {
		log.Fatalf("create %s error: %s", c.CreateType, "not in internal")
		return
	}

	f := createFile(filePath, strings.ToLower(c.FileName)+".go")
	if f == nil {
		log.Printf("warn: file %s%s %s", absLinuxPath, strings.ToLower(c.FileName)+".go", "already exists.")
		return
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	_, c.PackageName = filepath.Split(absPath)
	if c.PackageName == "" && c.CreateType == "service" {
		c.PackageName = "svc"
	} else if c.PackageName == "" {
		c.PackageName = c.CreateType
	}

	var t *template.Template
	var err error
	if tplPath == "" {
		t, err = template.ParseFS(tpl.CreateTplFS, fmt.Sprintf("create/%s.tpl", c.CreateType))
	} else {
		t, err = template.ParseFiles(path.Join(tplPath, fmt.Sprintf("%s.tpl", c.CreateType)))
	}
	if err != nil {
		log.Fatalf("create %s error: %s", c.CreateType, err.Error())
	}
	err = t.Execute(f, c)
	if err != nil {
		log.Fatalf("create %s error: %s", c.CreateType, err.Error())
	}
	log.Printf("Created new %s: %s", c.CreateType, absLinuxPath+strings.ToLower(c.FileName)+".go")

}

func createFile(dirPath string, filename string) *os.File {
	filePath := filepath.Join(dirPath, filename)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create dir %s: %v", dirPath, err)
	}
	stat, _ := os.Stat(filePath)
	if stat != nil {
		return nil
	}
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Failed to create file %s: %v", filePath, err)
	}

	return file
}
