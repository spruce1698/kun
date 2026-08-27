package create

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/spruce1698/kun/internal/create/kernel"
	"github.com/spruce1698/kun/pkg/helper"
	"github.com/spruce1698/kun/pkg/output"
	"github.com/spruce1698/kun/tpl"
)

const (
	BasePath = "internal"

	TypeHandler = "handler"
	TypeService = "service"
	TypeRouter  = "router"
	TypeCache   = "cache"
)

var (
	CmdCreate = &cobra.Command{
		Use:     "create [type] [name]",
		Short:   "Create a new hdl/svc/hs/rt/db/cache",
		Example: "kun create hdl user",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("create requires a subcommand: hdl, svc, hs, rt, db, cache")
		},
	}

	CmdCreateHandler = &cobra.Command{
		Use:     "hdl",
		Short:   "Create a new handler",
		Example: "kun create hdl user",
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

	CmdCreateHandlerAndService = &cobra.Command{
		Use:     "hs",
		Short:   "Create a new handler & service",
		Example: "kun create hs user",
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
		CmdCreateHandler, CmdCreateService, CmdCreateHandlerAndService,
		CmdCreateRouter, CmdCreateDBRepository, CmdCreateCacheRepository,
	} {
		c.Flags().StringP("tpl-path", "t", "", "template path")
	}
	for _, c := range []*cobra.Command{
		CmdCreateHandler, CmdCreateService, CmdCreateHandlerAndService,
		CmdCreateRouter, CmdCreateCacheRepository,
	} {
		c.Flags().BoolP("force", "f", false, "force override existing file")
	}
}

// Register E6: 将 create 及其子命令挂载到 parent，由本包自行维护命令树。
func Register(parent *cobra.Command) {
	parent.AddCommand(CmdCreate)
	CmdCreate.AddCommand(CmdCreateHandler)
	CmdCreate.AddCommand(CmdCreateService)
	CmdCreate.AddCommand(CmdCreateHandlerAndService)
	CmdCreate.AddCommand(CmdCreateRouter)
	CmdCreate.AddCommand(CmdCreateDBRepository)
	CmdCreate.AddCommand(CmdCreateCacheRepository)
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
	// importMarker 非默认包名时,import 注入所锚定的 marker 注释行(位于 DI 文件 import 块内)。
	importMarker string
	diBuilder    func(*Create) map[string]string
}

// 生成配置映射
var genConfigs = map[string]genConfig{
	TypeHandler: {
		typePath:     TypeHandler,
		defaultPkg:   TypeHandler,
		structSuffix: "Handler",
		importMarker: "// ==== Add Handler import before this line, don't edit this line.====",
		diBuilder: func(c *Create) map[string]string {
			packageName := c.PackageName + "."
			tPrefix := strings.ToUpper(string(c.PackageName[0])) + c.PackageName[1:]
			if c.PackageName == c.CreateType {
				packageName = ""
				tPrefix = ""
			}
			return map[string]string{
				"// ==== Add Handler to Ctx before this line, don't edit this line.====":     "\t" + tPrefix + c.FileName + "Handler *" + packageName + c.FileName + "Handler",
				"// ==== Add Handler to WireSet before this line, don't edit this line.====": "\twire.Struct(new(" + packageName + c.FileName + "Handler), \"*\"),",
			}
		},
	},
	TypeService: {
		typePath:     TypeService + "/svc",
		defaultPkg:   "svc",
		structSuffix: "Svc",
		importMarker: "// ==== Add Svc import before this line, don't edit this line.====",
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
		importMarker: "// ==== Add Rt import  before this line, don't edit this line.====",
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
		importMarker: "// ==== Add Repo import before this line, don't edit this line.====",
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

	c.CmdType = cmd.Name()
	arg := args[0]
	if c.CmdType == "svc" || c.CmdType == "hs" {
		if strings.HasPrefix(strings.ToLower(arg), "svc/") {
			arg = arg[4:]
		} else if strings.HasPrefix(strings.ToLower(arg), `svc\`) {
			arg = arg[4:]
		}
	}
	c.FilePath, c.FileName = filepath.Split(arg)
	cleanName := strings.TrimSuffix(c.FileName, ".go")
	cleanName = strings.TrimSpace(cleanName)
	if cleanName == "" {
		return fmt.Errorf("name argument %q cannot be empty or resolve to an empty name", arg)
	}
	runes := []rune(cleanName)
	c.FileName = string(unicode.ToUpper(runes[0])) + string(runes[1:])
	c.FileNameTitleLower = string(unicode.ToLower(runes[0])) + string(runes[1:])
	c.FileNameFirstChar = string(unicode.ToLower(runes[0]))

	switch c.CmdType {
	case "hdl":
		c.CreateType = TypeHandler
		return c.generateFile()

	case "svc":
		c.CreateType = TypeService
		return c.generateFile()

	case "hs":
		c.CreateType = TypeHandler
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

	// E2: 统一使用 filepath，再用 filepath.ToSlash 转正斜杠供展示
	// 构建文件路径
	var filePath string
	if c.FilePath == "" {
		filePath = filepath.ToSlash(filepath.Join(BasePath, config.typePath))
	} else {
		filePath = filepath.ToSlash(filepath.Join(BasePath, config.typePath, c.FilePath))
	}
	filePath = strings.TrimSuffix(filePath, "/")

	absPath, err := filepath.Abs(filepath.Join(filePath, fileName))
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	absDir := filepath.Dir(absPath)

	// 校验生成路径确实落在项目的 internal/<typePath> 之下,避免 c.FilePath 里的
	// "../" 把文件写到项目外。
	// 必须用 filepath.Rel 判断:子串匹配可被绕过 ——
	// "../../../tmp/internal/handler/x" 同样包含 "/internal/handler/"。
	expectedRoot, err := filepath.Abs(filepath.Join(BasePath, config.typePath))
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	rel, err := filepath.Rel(expectedRoot, absDir)
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return fmt.Errorf("create %s error: target path %s escapes %s", c.CreateType, absDir, expectedRoot)
	}

	absLinuxPath := filepath.ToSlash(absDir) + "/"

	// 计算相对 internal/ 的层级，用于模板中的 ../ 引用（如 mockgen 目标路径）。
	c.AddUPPath = strings.Repeat("../", strings.Count(filepath.ToSlash(c.FilePath), "/"))

	// 设置包名:取目标目录名(如 handler/service),而非文件名。
	c.PackageName = filepath.Base(absDir)
	if c.PackageName == "" {
		c.PackageName = config.defaultPkg
	}

	// 根据模板生成文件
	var t *template.Template
	if c.TplPath == "" {
		// E2: embed FS 路径使用正斜杠字符串拼接（path/filepath 在 Windows 下会用反斜杠）
		t, err = template.ParseFS(tpl.CreateTplFS, "create/"+c.CreateType+".tpl")
	} else {
		t, err = template.ParseFiles(filepath.Join(c.TplPath, c.CreateType+".tpl"))
	}
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	f, existed, err := createFile(filePath, fileName, c.Force)
	if err != nil {
		return fmt.Errorf("create %s error: %w", c.CreateType, err)
	}
	if existed {
		output.Warn("warn: file %s%s already exists.", absLinuxPath, fileName)
		return nil
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if err = t.Execute(f, c); err != nil {
		_ = f.Close()
		_ = os.Remove(filepath.Join(filePath, fileName))
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
	// E4: DI 文件查找规则——所有类型（含 cache）统一传"生成目录"，
	// 由 Wire2DIFile 向上逐级查找含 marker 的 DI 文件。
	// cache 的 DI 文件在 repository/（生成目录 repository/cache 的父级），
	// Wire2DIFile 的向上搜索逻辑可以自动找到，无需在此区分。
	diPath := absLinuxPath

	contentMap := config.diBuilder(c)
	// 非默认包名(即在子目录下生成)时,需要把新包的 import 注入到对应 DI 文件的 import 块。
	if c.PackageName != config.defaultPkg && config.importMarker != "" {
		contentMap[config.importMarker] = "\t\"" + c.ProjectName + "/" + filePath + "\""
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
	// M2: 目录权限改为 0755，避免 os.ModePerm(0777) 过于宽泛
	if err := os.MkdirAll(dirPath, 0755); err != nil {
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

// keysFileTpl E7: 用 text/template 生成 keys.go，避免字符串拼接在特殊字符场景下生成非法代码。
var keysFileTpl = template.Must(template.New("keys").Parse(
	`package {{.PackageName}}

const (
	{{.KeyName}} = "{{.KeyValue}}"
)
`))

// keysAppendTpl 用于向已存在的 const 块追加常量。
var keysAppendTpl = template.Must(template.New("keysAppend").Parse(
	`const (
	{{.KeyName}} = "{{.KeyValue}}"
`))

type keysData struct {
	PackageName string
	KeyName     string
	KeyValue    string
}

func generateKeysFile(dirPath string, c *Create) error {
	keysPath := filepath.Join(dirPath, "keys.go")
	keyName := c.FileName + "DataKey"
	keyValue := "cache:" + c.FileNameTitleLower + ":%d"

	kd := keysData{
		PackageName: c.PackageName,
		KeyName:     keyName,
		KeyValue:    keyValue,
	}

	// 1. 如果 keys.go 不存在，直接用模板创建
	if _, err := os.Stat(keysPath); os.IsNotExist(err) {
		var buf bytes.Buffer
		if err := keysFileTpl.Execute(&buf, kd); err != nil {
			return err
		}
		// M3: 统一文件权限 0644
		return os.WriteFile(keysPath, buf.Bytes(), 0644)
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

	// 4. 追加逻辑：向已有 const 块插入，或追加新 const 块
	if strings.Contains(content, "const (") {
		// 在已有 const ( 后插入新常量
		newConst := "\t" + keyName + " = \"" + keyValue + "\""
		content = strings.Replace(content, "const (", "const (\n"+newConst, 1)
	} else {
		// 追加新 const 块
		var buf bytes.Buffer
		if err := keysAppendTpl.Execute(&buf, kd); err != nil {
			return err
		}
		content += "\n" + buf.String() + ")\n"
	}

	return os.WriteFile(keysPath, []byte(content), 0644)
}
