package sqlgen

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/tools/imports"
	"gorm.io/gorm"
)

// Config generator's basic configuration
type Config struct {
	DbConn *gorm.DB // db connection

	OutPath     string // query code path
	PackageName string // generated model code's package name

	// generate model global configuration
	FieldNullable     bool // generate pointer when field is nullable
	FieldCoverable    bool // generate pointer when field has default value, to fix problem zero value cannot be assign: https://gorm.io/docs/create.html#Default-Values
	FieldSignable     bool // detect integer field's unsigned type, adjust generated data type
	FieldWithIndexTag bool // generate with gorm index tag
	FieldWithTypeTag  bool // generate with gorm column type tag
}

type StructMeta struct {
	DbConn         *gorm.DB
	FileName       string // generated file name
	InterfaceName  string // interface name
	StructName     string // origin/model struct name
	TableName      string // table name in db server
	PackageName    string
	PrimaryKeyType string // 主键key类型
	Fields         []*Field
}

// Field user input structures
type Field struct {
	Name         string
	Type         string
	GORMTag      string
	JSONTag      string
	CommentTag   string
	IsPrimaryKey bool
}

type Column struct {
	gorm.ColumnType
	TableName   string         `gorm:"column:TABLE_NAME"`
	Indexes     []*ColumnIndex `gorm:"-"`
	UseScanType bool           `gorm:"-"`
}

// Index table index info
type ColumnIndex struct {
	gorm.Index
	Priority int32 `gorm:"column:SEQ_IN_INDEX"`
}

type dataTypeMap map[string]func(detailType string) (finalType string)

var (
	defaultDataType             = "string"
	dataType        dataTypeMap = map[string]func(detailType string) (finalType string){
		"numeric":    func(string) string { return "int64" },
		"integer":    func(string) string { return "int64" },
		"int":        func(string) string { return "int64" },
		"smallint":   func(string) string { return "int64" },
		"mediumint":  func(string) string { return "int64" },
		"bigint":     func(string) string { return "int64" },
		"float":      func(string) string { return "float64" },
		"real":       func(string) string { return "float64" },
		"double":     func(string) string { return "float64" },
		"decimal":    func(string) string { return "float64" },
		"char":       func(string) string { return "string" },
		"varchar":    func(string) string { return "string" },
		"tinytext":   func(string) string { return "string" },
		"mediumtext": func(string) string { return "string" },
		"longtext":   func(string) string { return "string" },
		"binary":     func(string) string { return "[]byte" },
		"varbinary":  func(string) string { return "[]byte" },
		"tinyblob":   func(string) string { return "[]byte" },
		"blob":       func(string) string { return "[]byte" },
		"mediumblob": func(string) string { return "[]byte" },
		"longblob":   func(string) string { return "[]byte" },
		"text":       func(string) string { return "string" },
		"json":       func(string) string { return "string" },
		"enum":       func(string) string { return "string" },
		"time":       func(string) string { return "time.Time" },
		"date":       func(string) string { return "time.Time" },
		"datetime":   func(string) string { return "time.Time" },
		"timestamp":  func(string) string { return "time.Time" },
		"year":       func(string) string { return "int64" },
		"bit":        func(string) string { return "[]uint8" },
		"boolean":    func(string) string { return "bool" },
		"tinyint": func(detailType string) string {
			if strings.HasPrefix(strings.TrimSpace(detailType), "tinyint(1)") {
				return "bool"
			}
			return "int32"
		},
	}
)

var concurrent = runtime.NumCPU()

func init() { runtime.GOMAXPROCS(runtime.NumCPU()) }

// Generator code generator
type Generator struct {
	Cfg    Config
	models map[string]*StructMeta // gen model data
}

// NewGenerator create a new generator
func NewGenerator(cfg Config) *Generator {
	return &Generator{
		Cfg:    cfg,
		models: make(map[string]*StructMeta),
	}
}

// GenerateModel catch table info from db, return a BaseStruct
func (g *Generator) GenerateModel(tableName string) {
	structName := g.Cfg.DbConn.NamingStrategy.SchemaName(tableName)

	meta, err := g.getStructMeta(tableName, structName)
	if err != nil {
		g.Cfg.DbConn.Logger.Error(context.Background(), "generate struct from table fail: %s", err)
		panic("generate struct fail")
	}
	if meta == nil {
		log.Println(fmt.Sprintf("ignore table <%s>", tableName))
		return
	}
	g.models[meta.StructName] = meta

	log.Println(fmt.Sprintf("got %d columns from table <%s>", len(meta.Fields), meta.TableName))
	return
}

// Execute generate code to output path
func (g *Generator) Execute() {
	log.Println("Start generating code.")

	if err := g.generateModelFile(); err != nil {
		g.Cfg.DbConn.Logger.Error(context.Background(), "generate model struct fail: %s", err)
		panic("generate model struct fail")
	}

	log.Println("Generate code done.")
}

// getStructMeta generate db model by table name
func (g *Generator) getStructMeta(tableName, structName string) (*StructMeta, error) {
	if tableName == "" {
		return nil, nil
	}
	if err := checkStructName(structName); err != nil {
		return nil, fmt.Errorf("model name %q is invalid: %w", structName, err)
	}

	fileName := string(unicode.ToLower(rune(structName[0]))) + structName[1:]

	columns, err := g.getTableColumns(tableName)
	if err != nil || len(columns) == 0 {
		return nil, err
	}

	primaryKeyType := "int64"
	fields := make([]*Field, 0, len(columns))
	for _, col := range columns {
		m := col.ToField(g.Cfg.FieldNullable, g.Cfg.FieldCoverable, g.Cfg.FieldSignable)
		if t, ok := col.ColumnType.ColumnType(); ok && !g.Cfg.FieldWithTypeTag { // remove type tag if FieldWithTypeTag == false
			m.GORMTag = strings.ReplaceAll(m.GORMTag, ";type:"+t, "")
		}
		m.Name = g.Cfg.DbConn.NamingStrategy.SchemaName(m.Name)
		m.Name = strings.Replace(m.Name, "ID", "Id", -1)
		if m.IsPrimaryKey {
			primaryKeyType = m.Type
		}
		// json 小驼峰
		m.JSONTag = strings.ToLower(m.Name[:1]) + m.Name[1:]

		fields = append(fields, m)
	}

	return &StructMeta{
		DbConn:         g.Cfg.DbConn,
		FileName:       fileName,
		InterfaceName:  fileName,
		StructName:     structName,
		TableName:      tableName,
		PackageName:    g.Cfg.PackageName,
		PrimaryKeyType: primaryKeyType,
		Fields:         fields,
	}, nil
}

func (g *Generator) getTableColumns(tableName string) (result []*Column, err error) {
	types, err := g.Cfg.DbConn.Migrator().ColumnTypes(tableName)
	if err != nil {
		return nil, err
	}

	for _, column := range types {
		result = append(result, &Column{
			ColumnType:  column,
			TableName:   tableName,
			UseScanType: g.Cfg.DbConn.Dialector.Name() != "mysql" && g.Cfg.DbConn.Dialector.Name() != "sqlite",
		})
	}

	if !g.Cfg.FieldWithIndexTag || len(result) == 0 {
		return result, nil
	}

	indexList, err := g.Cfg.DbConn.Migrator().GetIndexes(tableName)
	if err != nil { // ignore find index err
		g.Cfg.DbConn.Logger.Warn(context.Background(), "GetTableIndex for %s,err=%s", tableName, err.Error())
		return result, nil
	}
	if len(indexList) == 0 {
		return result, nil
	}

	columnIndexMap := make(map[string][]*ColumnIndex, len(indexList))
	for _, idx := range indexList {
		if idx == nil {
			continue
		}
		for i, col := range idx.Columns() {
			columnIndexMap[col] = append(columnIndexMap[col], &ColumnIndex{
				Index:    idx,
				Priority: int32(i + 1),
			})
		}
	}
	for _, c := range result {
		c.Indexes = columnIndexMap[c.Name()]
	}
	return result, nil
}

// generateModelFile generate model structures and save to file
func (g *Generator) generateModelFile() error {
	if len(g.models) == 0 {
		return nil
	}

	modelOutPath, err := g.getModelOutputPath()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(modelOutPath, os.ModePerm); err != nil {
		return fmt.Errorf("create model pkg path(%s) fail: %s", modelOutPath, err)
	}

	errChan := make(chan error)
	p := NewPool(concurrent)
	for _, data := range g.models {
		if data == nil {
			continue
		}
		p.Add()
		go func(data *StructMeta) {
			defer p.Done()

			var buf bytes.Buffer
			err := render(BaseDbMethod, &buf, data)
			if err != nil {
				errChan <- err
				return
			}

			modelFile := modelOutPath + data.FileName + "_gen.go"
			err = g.output(modelFile, buf.Bytes())
			if err != nil {
				errChan <- err
				return
			}

			log.Println(fmt.Sprintf("generate model file(table <%s> -> {%s.%s}): %s", data.TableName, data.PackageName, data.StructName, modelFile))

			modelFile = modelOutPath + data.FileName + ".go"
			_, StatErr := os.Stat(modelFile)
			if StatErr != nil {
				var buffer bytes.Buffer
				err = render(InterfaceMethod, &buffer, data)
				if err != nil {
					errChan <- err
					return
				}
				err = g.output(modelFile, buffer.Bytes())
				if err != nil {
					errChan <- err
					return
				}
				log.Println(fmt.Sprintf("generate model file(table <%s> -> {%s.%s}): %s", data.TableName, data.PackageName, data.StructName, modelFile))
			}
		}(data)
	}
	select {
	case err = <-errChan:
		return err
	case <-p.AsyncWaitAll():
	}
	return nil
}

func (g *Generator) getModelOutputPath() (outPath string, err error) {
	if strings.Contains(g.Cfg.PackageName, string(os.PathSeparator)) {
		outPath, err = filepath.Abs(g.Cfg.PackageName)
		if err != nil {
			return "", fmt.Errorf("cannot parse model pkg path: %w", err)
		}
	} else {
		outPath = filepath.Join(filepath.Dir(g.Cfg.OutPath), g.Cfg.PackageName)
	}
	return outPath + string(os.PathSeparator), nil
}

// output format and output
func (g *Generator) output(fileName string, content []byte) error {
	result, err := imports.Process(fileName, content, nil)
	if err != nil {
		lines := strings.Split(string(content), "\n")
		errLine, _ := strconv.Atoi(strings.Split(err.Error(), ":")[1])
		startLine, endLine := errLine-5, errLine+5
		fmt.Println("Format fail:", errLine, err)
		if startLine < 0 {
			startLine = 0
		}
		if endLine > len(lines)-1 {
			endLine = len(lines) - 1
		}
		for i := startLine; i <= endLine; i++ {
			fmt.Println(i, lines[i])
		}
		return fmt.Errorf("cannot format file: %w", err)
	}
	return ioutil.WriteFile(fileName, result, 0640)
}

func render(tmpl string, wr io.Writer, data interface{}) error {
	t, err := template.New(tmpl).Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(wr, data)
}

func checkStructName(name string) error {
	if name == "" {
		return nil
	}
	if !regexp.MustCompile(`^\w+$`).MatchString(name) {
		return fmt.Errorf("model name cannot contains invalid character")
	}
	if name[0] < 'A' || name[0] > 'Z' {
		return fmt.Errorf("model name must be initial capital")
	}
	return nil
}

func (m dataTypeMap) Get(dataType, detailType string) string {
	if convert, ok := m[strings.ToLower(dataType)]; ok {
		return convert(detailType)
	}
	return defaultDataType
}

func (c *Column) columnType() (v string) {
	if cl, ok := c.ColumnType.ColumnType(); ok {
		return cl
	}
	return c.DatabaseTypeName()
}

// needDefaultTag check if default tag needed
func (c *Column) needDefaultTag(defaultTagValue string) bool {
	if defaultTagValue == "" {
		return false
	}
	switch c.ScanType().Kind() {
	case reflect.Bool:
		return defaultTagValue != "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return defaultTagValue != "0"
	case reflect.String:
		return defaultTagValue != ""
	case reflect.Struct:
		return strings.Trim(defaultTagValue, "'0:- ") != ""
	}
	return c.Name() != "created_at" && c.Name() != "updated_at"
}

// defaultTagValue return gorm default tag's value
func (c *Column) defaultTagValue() string {
	value, ok := c.DefaultValue()
	if !ok {
		return ""
	}
	if value != "" && strings.TrimSpace(value) == "" {
		return "'" + value + "'"
	}
	return value
}

func (c *Column) buildGormTag() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("column:%s;type:%s", c.Name(), c.columnType()))

	isPriKey, ok := c.PrimaryKey()
	isValidPriKey := ok && isPriKey
	if isValidPriKey {
		buf.WriteString(";primaryKey")
		if at, ok := c.AutoIncrement(); ok {
			buf.WriteString(fmt.Sprintf(";autoIncrement:%t", at))
		}
	} else if n, ok := c.Nullable(); ok && !n {
		buf.WriteString(";not null")
	}

	for _, idx := range c.Indexes {
		if idx == nil {
			continue
		}
		if pk, _ := idx.PrimaryKey(); pk { // ignore PrimaryKey
			continue
		}
		if uniq, _ := idx.Unique(); uniq {
			buf.WriteString(fmt.Sprintf(";uniqueIndex:%s,priority:%d", idx.Name(), idx.Priority))
		} else {
			buf.WriteString(fmt.Sprintf(";index:%s,priority:%d", idx.Name(), idx.Priority))
		}
	}

	if dtValue := c.defaultTagValue(); !isValidPriKey && c.needDefaultTag(dtValue) { // cannot set default tag for primary key
		buf.WriteString(fmt.Sprintf(`;default:%s`, dtValue))
	}
	return buf.String()
}

// ToField convert to field
func (c *Column) ToField(nullable, coverable, signable bool) *Field {
	fieldType := ""
	if c.UseScanType && c.ScanType() != nil {
		fieldType = c.ScanType().String()
	} else {
		fieldType = dataType.Get(c.DatabaseTypeName(), c.columnType())
	}

	if signable && strings.Contains(c.columnType(), "unsigned") && strings.HasPrefix(fieldType, "int") {
		fieldType = "u" + fieldType
	}
	switch {
	case c.Name() == "deleted_at" && fieldType == "time.Time":
		fieldType = "gorm.DeletedAt"
	case coverable && c.needDefaultTag(c.defaultTagValue()):
		fieldType = "*" + fieldType
	case nullable:
		if n, ok := c.Nullable(); ok && n {
			fieldType = "*" + fieldType
		}
	}

	var commentTag string
	if ct, ok := c.Comment(); ok {
		commentTag = fmt.Sprintf("// %s", ct)
	}

	isPriKey, ok := c.PrimaryKey()
	isPrimaryKey := ok && isPriKey

	return &Field{
		Name:         c.Name(),
		Type:         fieldType,
		GORMTag:      c.buildGormTag(),
		JSONTag:      c.Name(),
		CommentTag:   commentTag,
		IsPrimaryKey: isPrimaryKey,
	}
}

// Tags ...
func (m *Field) Tags() string {
	var tags strings.Builder
	if gormTag := strings.TrimSpace(m.GORMTag); gormTag != "" {
		tags.WriteString(fmt.Sprintf(`gorm:"%s" `, gormTag))
	}
	if jsonTag := strings.TrimSpace(m.JSONTag); jsonTag != "" {
		tags.WriteString(fmt.Sprintf(`json:"%s" `, jsonTag))
	}
	return strings.TrimSpace(tags.String())
}

// StructComment struct comment
func (b *StructMeta) StructComment() string {
	if b.TableName != "" {
		return fmt.Sprintf(`mapped from table <%s>`, b.TableName)
	}
	return `mapped from object`
}
