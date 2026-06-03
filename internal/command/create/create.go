package create

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/internal/command/create/kernel"
	"github.com/spruce1698/kun/pkg/fmt"
	"github.com/spruce1698/kun/pkg/helper"
	"github.com/spruce1698/kun/tpl"
)

const (
	BasePath = "internal"

	TypeController = "controller"
	TypeService    = "service"
	TypeRouter     = "router"
	TypeCache      = "cache"
)

var (
	tplPath string

	CmdCreate = &cobra.Command{
		Use:     "create [type] [name]",
		Short:   "Create a new ctrl/svc/cs/rt/db/cache",
		Example: "kun create ctrl user",
		Args:    cobra.ExactArgs(2),
		Run:     func(cmd *cobra.Command, args []string) {},
	}

	CmdCreateController = &cobra.Command{
		Use:     "ctrl",
		Short:   "Create a new controller",
		Example: "kun create ctrl user",
		Args:    cobra.ExactArgs(1),
		Run:     runCreate,
	}

	CmdCreateService = &cobra.Command{
		Use:     "svc",
		Short:   "Create a new service",
		Example: "kun create svc user",
		Args:    cobra.ExactArgs(1),
		Run:     runCreate,
	}

	CmdCreateControllerAndService = &cobra.Command{
		Use:     "cs",
		Short:   "Create a new controller & service",
		Example: "kun create cs user",
		Args:    cobra.ExactArgs(1),
		Run:     runCreate,
	}

	CmdCreateRouter = &cobra.Command{
		Use:     "rt",
		Short:   "Create a new router",
		Example: "kun create rt user",
		Args:    cobra.ExactArgs(1),
		Run:     runCreate,
	}

	CmdCreateDBRepository = &cobra.Command{
		Use:     "db",
		Short:   "Create a new DB repository",
		Example: "kun create db \"name:pwd@tcp(127.0.0.1:3306)/dbname\" [t1,t2|t1|*]",
		Args:    cobra.ExactArgs(2),
		Run:     genDBRepo,
	}

	CmdCreateCacheRepository = &cobra.Command{
		Use:     "cache",
		Short:   "Create a new cache repository",
		Example: "kun create cache ",
		Args:    cobra.ExactArgs(1),
		Run:     runCreate,
	}
)

func init() {
	CmdCreateController.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateService.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateControllerAndService.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateRouter.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateDBRepository.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")

	CmdCreateCacheRepository.Flags().StringVarP(&tplPath, "tpl-path", "t", tplPath, "template path")
}

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

// 文件生成配置
type genConfig struct {
	typePath     string
	defaultPkg   string
	structSuffix string
	diBuilder    func(*Create) map[string]string
}

// 生成配置映射
var genConfigs = map[string]genConfig{
	TypeController: {
		typePath:     TypeController,
		defaultPkg:   TypeController,
		structSuffix: "Ctrl",
		diBuilder: func(c *Create) map[string]string {
			packageName := c.PackageName + "."
			tPrefix := strings.ToUpper(string(c.PackageName[0])) + c.PackageName[1:]
			if c.PackageName == c.CreateType {
				packageName = ""
				tPrefix = ""
			}
			return map[string]string{
				"// ==== Add CtrlCtx before this line, don't edit this line.====": "\t" + tPrefix + c.FileName + "Ctrl *" + packageName + c.FileName + "Ctrl",
				"// ==== Add Ctrl before this line, don't edit this line.====":    "\twire.Struct(new(" + packageName + c.FileName + "Ctrl), \"*\"),",
			}
		},
	},
	TypeService: {
		typePath:     TypeService + "/svc",
		defaultPkg:   "svc",
		structSuffix: "Svc",
		diBuilder: func(c *Create) map[string]string {
			return map[string]string{
				"// ==== Add Svc before this line, don't edit this line.====": "\twire.Struct(new(" + c.PackageName + "." + c.FileName + "Ctx), \"*\"),\n    " +
					c.PackageName + ".New" + c.FileName + "Svc,",
			}
		},
	},
	TypeRouter: {
		typePath:     TypeRouter,
		defaultPkg:   TypeRouter,
		structSuffix: "",
		diBuilder: func(c *Create) map[string]string {
			packageName := c.PackageName + "."
			if c.PackageName == c.CreateType {
				packageName = ""
			}
			return map[string]string{
				"// ==== Add Rt before this line, don't edit this line.====": "\t\t" + packageName + c.FileName + ",",
			}
		},
	},
	TypeCache: {
		typePath:     "repository/cache",
		defaultPkg:   "cache",
		structSuffix: "Cache",
		diBuilder: func(c *Create) map[string]string {
			return map[string]string{
				"// ==== Add Repo before this line, don't edit this line.====": "\t" + c.PackageName + ".New" + c.FileName + "Cache,",
			}
		},
	},
}

func runCreate(cmd *cobra.Command, args []string) {
	c := NewCreate()
	c.ProjectName = helper.GetProjectName(".")
	if c.ProjectName == "" {
		return
	}

	c.CmdType = cmd.Use
	c.FilePath, c.FileName = filepath.Split(args[0])
	c.FileName = strings.ReplaceAll(strings.ToUpper(string(c.FileName[0]))+c.FileName[1:], ".go", "")
	c.FileNameTitleLower = strings.ToLower(string(c.FileName[0])) + c.FileName[1:]
	c.FileNameFirstChar = string(c.FileNameTitleLower[0])

	switch c.CmdType {
	case "ctrl":
		c.CreateType = TypeController
		c.generateFile()

	case "svc":
		c.CreateType = TypeService
		c.generateFile()

	case "cs":
		c.CreateType = TypeController
		c.generateFile()

		c.CreateType = TypeService
		c.generateFile()

	case "rt":
		c.CreateType = TypeRouter
		c.generateFile()

	case "cache":
		c.CreateType = TypeCache
		c.generateFile()

	default:
		fmt.Error("Invalid type: %s", c.CmdType)
	}

}

func (c *Create) generateFile() {
	config, ok := genConfigs[c.CreateType]
	if !ok {
		fmt.Error("Invalid type: %s", c.CmdType)
		return
	}

	fileName := strings.ToLower(string(c.FileName[0])) + c.FileName[1:] + ".go"

	// 构建文件路径
	filePath := c.FilePath
	if filePath == "" {
		filePath = filepath.Join(BasePath, config.typePath)
	} else {
		c.AddUPPath = strings.Repeat("../", strings.Count(filePath, "/"))
		filePath = filepath.Join(BasePath, config.typePath, filePath)
	}
	filePath = strings.ReplaceAll(strings.ReplaceAll(filePath+"/", "//", "/"), "\\", "/")

	absPath, _ := filepath.Abs(filepath.Dir(filepath.Join(filePath, fileName)))
	absLinuxPath := strings.ReplaceAll(absPath, "\\", "/") + "/"
	if strings.LastIndex(absLinuxPath, filePath) < 1 {
		fmt.Error("create %s error: %s", c.CreateType, "not in internal")
		return
	}

	// 设置包名
	_, c.PackageName = filepath.Split(absPath)
	if c.PackageName == "" {
		c.PackageName = config.defaultPkg
	}

	// 根据模板生成文件
	var t *template.Template
	var err error
	if tplPath == "" {
		t, err = template.ParseFS(tpl.CreateTplFS, fmt.Sprintf("create/%s.tpl", c.CreateType))
	} else {
		t, err = template.ParseFiles(path.Join(tplPath, fmt.Sprintf("%s.tpl", c.CreateType)))
	}
	if err != nil {
		fmt.Error("create %s error: %s", c.CreateType, err)
		return
	}
	f := createFile(filePath, fileName)
	if f == nil {
		fmt.Warn("warn: file %s%s %s", absLinuxPath, fileName, "already exists.")
		return
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	err = t.Execute(f, c)
	if err != nil {
		fmt.Error("create %s error: %s", c.CreateType, err)
		return
	}
	fmt.Success("created new %s: %s", c.CreateType, filepath.Join(absLinuxPath, fileName))

	// 更新DI文件
	diPath := absLinuxPath
	if c.CreateType != TypeCache {
		diPath, _ = filepath.Abs(filepath.Dir(filepath.Join(BasePath, c.CreateType, "X/X")))
		diPath = strings.ReplaceAll(diPath, "\\", "/")
	}

	contentMap := config.diBuilder(c)
	if c.PackageName != config.defaultPkg {
		contentMap["github.com/google/wire"] = "\t\"" + c.ProjectName + "/" + strings.TrimRight(filePath, "/") + "\""
		contentMap["// ==== Add Rt import  before this line, don't edit this line.===="] = "\t\"" + c.ProjectName + "/" + strings.TrimRight(filePath, "/") + "\""
	}

	if err = kernel.Wire2DIFile(diPath, contentMap); err != nil {
		fmt.Error("generate insert New%s%s to DI file error: %s", c.FileName, config.structSuffix, err)
		return
	}
	fmt.Success("generate insert New%s%s to DI file", c.FileName, config.structSuffix)
}

func createFile(dirPath string, filename string) *os.File {
	filePath := filepath.Join(dirPath, filename)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		fmt.Error("failed to create dir %s: %v", dirPath, err)
	}
	stat, _ := os.Stat(filePath)
	if stat != nil {
		return nil
	}
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Error("failed to create file %s: %v", filePath, err)
	}

	return file
}
