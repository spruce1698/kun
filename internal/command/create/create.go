package create

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/internal/command/create/kernel"
	"github.com/spruce1698/kun/pkg/helper"
	"github.com/spruce1698/kun/pkg/output"
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
	CmdCreate = &cobra.Command{
		Use:     "create [type] [name]",
		Short:   "Create a new ctrl/svc/cs/rt/db/cache",
		Example: "kun create ctrl user",
		Args:    cobra.ExactArgs(2),
		RunE:    func(cmd *cobra.Command, args []string) error { return nil },
	}

	CmdCreateController = &cobra.Command{
		Use:     "ctrl",
		Short:   "Create a new controller",
		Example: "kun create ctrl user",
		Args:    cobra.ExactArgs(1),
		RunE:    runCreate,
	}

	CmdCreateService = &cobra.Command{
		Use:     "svc",
		Short:   "Create a new service",
		Example: "kun create svc user",
		Args:    cobra.ExactArgs(1),
		RunE:    runCreate,
	}

	CmdCreateControllerAndService = &cobra.Command{
		Use:     "cs",
		Short:   "Create a new controller & service",
		Example: "kun create cs user",
		Args:    cobra.ExactArgs(1),
		RunE:    runCreate,
	}

	CmdCreateRouter = &cobra.Command{
		Use:     "rt",
		Short:   "Create a new router",
		Example: "kun create rt user",
		Args:    cobra.ExactArgs(1),
		RunE:    runCreate,
	}

	CmdCreateDBRepository = &cobra.Command{
		Use:     "db [DSN|SQL_FILE] [tables|*]",
		Short:   "Create a new DB repository from DB connection or SQL file",
		Example: `kun create db "name:pwd@tcp(127.0.0.1:3306)/dbname" [t1,t2|t1|*]  OR  kun create db "schema.sql" [t1,t2|*]`,
		Args:    cobra.RangeArgs(1, 2),
		RunE:    genDBRepo,
	}

	CmdCreateCacheRepository = &cobra.Command{
		Use:     "cache",
		Short:   "Create a new cache repository",
		Example: "kun create cache ",
		Args:    cobra.ExactArgs(1),
		RunE:    runCreate,
	}
)

func init() {
	// flag 绑定到各子命令;业务逻辑通过 cmd.Flags().Get* 读取,避免包级变量在多次调用间泄漏。
	for _, c := range []*cobra.Command{
		CmdCreateController, CmdCreateService, CmdCreateControllerAndService,
		CmdCreateRouter, CmdCreateDBRepository, CmdCreateCacheRepository,
	} {
		c.Flags().StringP("tpl-path", "t", "", "template path")
	}
	for _, c := range []*cobra.Command{
		CmdCreateController, CmdCreateService, CmdCreateControllerAndService,
		CmdCreateRouter, CmdCreateCacheRepository,
	} {
		c.Flags().BoolP("force", "f", false, "force override existing file")
	}
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
	TplPath            string
	Force              bool
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

func runCreate(cmd *cobra.Command, args []string) error {
	c := NewCreate()
	projectName, err := helper.GetProjectName(".")
	if err != nil {
		return fmt.Errorf("get project name error: %w", err)
	}
	c.ProjectName = projectName
	// 从当前命令的 flag 读取，避免全局变量在多次调用间泄漏
	c.TplPath, _ = cmd.Flags().GetString("tpl-path")
	c.Force, _ = cmd.Flags().GetBool("force")

	c.CmdType = cmd.Use
	arg := args[0]
	if c.CmdType == "svc" || c.CmdType == "cs" {
		if strings.HasPrefix(strings.ToLower(arg), "svc/") {
			arg = arg[4:]
		} else if strings.HasPrefix(strings.ToLower(arg), "svc\\") {
			arg = arg[4:]
		}
	}
	c.FilePath, c.FileName = filepath.Split(arg)
	c.FileName = strings.ReplaceAll(strings.ToUpper(string(c.FileName[0]))+c.FileName[1:], ".go", "")
	c.FileNameTitleLower = strings.ToLower(string(c.FileName[0])) + c.FileName[1:]
	c.FileNameFirstChar = string(c.FileNameTitleLower[0])

	switch c.CmdType {
	case "ctrl":
		c.CreateType = TypeController
		return c.generateFile()

	case "svc":
		c.CreateType = TypeService
		return c.generateFile()

	case "cs":
		c.CreateType = TypeController
		if err := c.generateFile(); err != nil {
			return err
		}
		c.CreateType = TypeService
		return c.generateFile()

	case "rt":
		c.CreateType = TypeRouter
		return c.generateFile()

	case "cache":
		c.CreateType = TypeCache
		return c.generateFile()

	default:
		return fmt.Errorf("invalid type: %s", c.CmdType)
	}

}

func (c *Create) generateFile() error {
	config, ok := genConfigs[c.CreateType]
	if !ok {
		return fmt.Errorf("invalid type: %s", c.CmdType)
	}

	fileName := strings.ToLower(string(c.FileName[0])) + c.FileName[1:] + ".go"

	// 构建文件路径(统一用正斜杠,跨平台一致)
	filePath := c.FilePath
	if filePath == "" {
		filePath = filepath.ToSlash(filepath.Join(BasePath, config.typePath))
	} else {
		filePath = filepath.ToSlash(filepath.Join(BasePath, config.typePath, filePath))
	}
	filePath = strings.TrimSuffix(filePath, "/")

	absPath, err := filepath.Abs(filepath.Join(filePath, fileName))
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	absDir := filepath.Dir(absPath)
	absLinuxPath := filepath.ToSlash(absDir) + "/"

	// 校验生成路径确实落在项目的 internal/<typePath> 下,避免误生成到外部目录。
	// 用 ToSlash 统一后检查路径段,而非依赖脆弱的子串 LastIndex。
	expectedSeg := "/internal/" + config.typePath
	if !strings.Contains(absLinuxPath, expectedSeg) {
		return fmt.Errorf("create %s error: target path %s is not under %s", c.CreateType, absLinuxPath, expectedSeg)
	}

	// 计算相对 internal/ 的层级，用于模板中的 ../ 引用（如 mockgen 目标路径）。
	// 深度 = 自定义子路径（c.FilePath）的路径段数；默认路径时为 0，模板已含固定 ../../../。
	c.AddUPPath = strings.Repeat("../", strings.Count(filepath.ToSlash(c.FilePath), "/"))

	// 设置包名:取目标目录名(如 controller/service),而非文件名。
	c.PackageName = filepath.Base(absDir)
	if c.PackageName == "" {
		c.PackageName = config.defaultPkg
	}

	// 根据模板生成文件
	var t *template.Template
	if c.TplPath == "" {
		t, err = template.ParseFS(tpl.CreateTplFS, fmt.Sprintf("create/%s.tpl", c.CreateType))
	} else {
		t, err = template.ParseFiles(path.Join(c.TplPath, fmt.Sprintf("%s.tpl", c.CreateType)))
	}
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	f, existed, err := createFile(filePath, fileName, c.Force)
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	if existed {
		output.Warn("warn: file %s%s %s", absLinuxPath, fileName, "already exists.")
		return nil
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if err = t.Execute(f, c); err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	output.Success("created new %s: %s", c.CreateType, filepath.Join(absLinuxPath, fileName))

	if c.CreateType == TypeCache {
		if err = generateKeysFile(absLinuxPath, c); err != nil {
			return fmt.Errorf("generate keys.go error: %w", err)
		}
		output.Success("generate keys.go in %s", absLinuxPath)
	}

	// 更新DI文件
	// DI 文件位置:cache 的 DI 文件在 repository/(生成目录 repository/cache 的父级),
	// 其余类型的 DI 文件就在生成目录本身(controller/service-svc/router)。
	// 统一传"生成目录",由 Wire2DIFile 向上逐级查找 marker DI 文件。
	diPath := absLinuxPath
	if c.CreateType != TypeCache {
		diPath = filepath.ToSlash(filepath.Join(BasePath, config.typePath))
	}

	contentMap := config.diBuilder(c)
	if c.PackageName != config.defaultPkg {
		contentMap["github.com/google/wire"] = "\t\"" + c.ProjectName + "/" + filePath + "\""
		contentMap["// ==== Add Rt import  before this line, don't edit this line.===="] = "\t\"" + c.ProjectName + "/" + filePath + "\""
	}

	if err = kernel.Wire2DIFile(diPath, contentMap); err != nil {
		return fmt.Errorf("generate insert New%s%s to DI file error: %w", c.FileName, config.structSuffix, err)
	}
	output.Success("generate insert New%s%s to DI file", c.FileName, config.structSuffix)
	return nil
}

// createFile 创建文件。返回 (file, existed, err)：existed 表示文件已存在且未强制覆盖。
func createFile(dirPath string, filename string, force bool) (*os.File, bool, error) {
	filePath := filepath.Join(dirPath, filename)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create dir %s: %w", dirPath, err)
	}
	if stat, err := os.Stat(filePath); err == nil && stat != nil && !force {
		return nil, true, nil
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	return file, false, nil
}

func generateKeysFile(dirPath string, c *Create) error {
	keysPath := filepath.Join(dirPath, "keys.go")
	keyName := c.FileName + "DataKey"
	keyValue := "cache:" + c.FileNameTitleLower + ":%d"

	// 1. 如果 keys.go 不存在，直接创建并写入初始模板
	if _, err := os.Stat(keysPath); os.IsNotExist(err) {
		content := "package " + c.PackageName + "\n\nconst (\n\t" + keyName + " = \"" + keyValue + "\"\n)\n"
		return os.WriteFile(keysPath, []byte(content), 0644)
	}

	// 2. 如果 keys.go 已存在，读取它
	data, err := os.ReadFile(keysPath)
	if err != nil {
		return err
	}
	content := string(data)

	// 3. 幂等：若已经包含此常量的名称，则什么都不做
	if strings.Contains(content, keyName) {
		return nil
	}

	// 4. 追加逻辑
	if strings.Contains(content, "const (") {
		content = strings.Replace(content, "const (", "const (\n\t"+keyName+" = \""+keyValue+"\"", 1)
	} else {
		content += "\n\nconst (\n\t" + keyName + " = \"" + keyValue + "\"\n)\n"
	}

	return os.WriteFile(keysPath, []byte(content), 0644)
}
