package kernel

import (
    "bytes"
    "os"
    "path/filepath"
    "reflect"
    "regexp"
    "runtime"
    "strings"
    "text/template"
    "unicode"
    
    "github.com/spruce1698/kun/pkg/fmt"
    "github.com/spruce1698/kun/tpl"
    "golang.org/x/tools/imports"
    "gorm.io/gorm"
)

// generator's basic configuration
type SQLConfig struct {
    DbConn *gorm.DB // db connection
    
    OutPath     string // query code path
    PackageName string // generated repository code's package name
    
    // generate repository global configuration
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
    StructName     string // origin/repo struct name
    TableName      string // table name in db server
    PackageName    string
    PrimaryKeyType string // 主键key类型
    Fields         []*Field
}

//  user input structures
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

func init() { runtime.GOMAXPROCS(runtime.NumCPU()) }

// code generator
type Generator struct {
    Conf  SQLConfig
    repos map[string]*StructMeta // gen repository data
}

//  create a new generator
func NewGenerator(conf SQLConfig) *Generator {
    return &Generator{
        Conf:  conf,
        repos: make(map[string]*StructMeta),
    }
}

// catch table info from db, return a BaseStruct
func (g *Generator) GenerateRepo(tableName string) {
    structName := g.Conf.DbConn.NamingStrategy.SchemaName(tableName)
    
    meta, err := g.getStructMeta(tableName, structName)
    if err != nil {
        fmt.Error("generate struct from table fail: %s", err)
        return
    }
    if meta == nil {
        fmt.Success("ignore table <%s>", tableName)
        return
    }
    g.repos[meta.StructName] = meta
    
    fmt.Success("got %d columns from table <%s>", len(meta.Fields), meta.TableName)
    return
}

// Execute generate code to output path
func (g *Generator) Execute() {
    fmt.Success("Start generating code.")
    
    if err := g.generateRepoFile(); err != nil {
        fmt.Error("generate repository struct fail: %s", err)
        return
    }
    
    fmt.Success("Generate code done.")
}

// generate db repository by table name
func (g *Generator) getStructMeta(tableName, structName string) (*StructMeta, error) {
    if tableName == "" {
        return nil, nil
    }
    if err := g.checkStructName(structName); err != nil {
        return nil, fmt.Errorf("repo name %q is invalid: %w", structName, err)
    }
    
    fileName := string(unicode.ToLower(rune(structName[0]))) + structName[1:]
    
    columns, err := g.getTableColumns(tableName)
    if err != nil || len(columns) == 0 {
        return nil, err
    }
    
    primaryKeyType := "int64"
    fields := make([]*Field, 0, len(columns))
    for _, col := range columns {
        m := col.ToField(g.Conf.FieldNullable, g.Conf.FieldCoverable, g.Conf.FieldSignable)
        if t, ok := col.ColumnType.ColumnType(); ok && !g.Conf.FieldWithTypeTag { // remove type tag if FieldWithTypeTag == false
            m.GORMTag = strings.ReplaceAll(m.GORMTag, ";type:"+t, "")
        }
        m.Name = g.Conf.DbConn.NamingStrategy.SchemaName(m.Name)
        m.Name = strings.Replace(m.Name, "ID", "Id", -1)
        if m.IsPrimaryKey {
            primaryKeyType = m.Type
        }
        // json 小驼峰
        m.JSONTag = strings.ToLower(m.Name[:1]) + m.Name[1:]
        
        fields = append(fields, m)
    }
    
    return &StructMeta{
        DbConn:         g.Conf.DbConn,
        FileName:       fileName,
        InterfaceName:  fileName,
        StructName:     structName,
        TableName:      tableName,
        PackageName:    g.Conf.PackageName,
        PrimaryKeyType: primaryKeyType,
        Fields:         fields,
    }, nil
}

func (g *Generator) getTableColumns(tableName string) (result []*Column, err error) {
    types, err := g.Conf.DbConn.Migrator().ColumnTypes(tableName)
    if err != nil {
        return nil, err
    }
    
    for _, column := range types {
        result = append(result, &Column{
            ColumnType:  column,
            TableName:   tableName,
            UseScanType: g.Conf.DbConn.Dialector.Name() != "mysql" && g.Conf.DbConn.Dialector.Name() != "sqlite",
        })
    }
    
    if !g.Conf.FieldWithIndexTag || len(result) == 0 {
        return result, nil
    }
    
    indexList, err := g.Conf.DbConn.Migrator().GetIndexes(tableName)
    if err != nil { // ignore find index err
        fmt.Warn("GetTableIndex for %s,err=%s", tableName, err.Error())
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

// generate repository structures and save to file
func (g *Generator) generateRepoFile() error {
    if len(g.repos) == 0 {
        return nil
    }
    
    repoOutPath, err := g.getRepoOutputPath()
    if err != nil {
        return err
    }
    
    if err = os.MkdirAll(repoOutPath, os.ModePerm); err != nil {
        return fmt.Errorf("create repository pkg path(%s) fail: %w", repoOutPath, err)
    }
    
    for _, data := range g.repos {
        if data == nil {
            continue
        }
        repoFile := filepath.Join(repoOutPath, data.FileName+"_gen.go")
        err = g.output("dbDefault", data, repoFile)
        if err != nil {
            return err
        }
        
        fmt.Success("generate repository file(table <%s> -> {%s.%s}): %s", data.TableName, data.PackageName, data.StructName, repoFile)
        
        repoFile = filepath.Join(repoOutPath, data.FileName+".go")
        _, StatErr := os.Stat(repoFile)
        if StatErr != nil {
            err = g.output("dbCustom", data, repoFile)
            if err != nil {
                return err
            }
            fmt.Success("generate repository file(table <%s> -> {%s.%s}): %s", data.TableName, data.PackageName, data.StructName, repoFile)
        }
        
        contentMap := map[string]string{
            "// ==== Add Repo before this line, don't edit this line.====": "    " + data.PackageName + ".New" + data.StructName + "Db,",
        }
        err = Wire2DIFile(repoOutPath, contentMap)
        if err != nil {
            fmt.Error("generate db repository insert New%sDb to DI file error: %s", data.StructName, err)
            continue
        }
        fmt.Success("generate db repository insert New%sDb to DI file", data.StructName)
    }
    
    return nil
}

func (g *Generator) getRepoOutputPath() (outPath string, err error) {
    if strings.Contains(g.Conf.PackageName, string(os.PathSeparator)) {
        outPath, err = filepath.Abs(g.Conf.PackageName)
        if err != nil {
            return "", fmt.Errorf("cannot parse repository pkg path: %w", err)
        }
    } else {
        outPath = filepath.Join(filepath.Dir(g.Conf.OutPath), g.Conf.PackageName)
    }
    return outPath + string(os.PathSeparator), nil
}

// format and output
func (g *Generator) output(tmpl string, data interface{}, fileName string) error {
    t, err := template.ParseFS(tpl.CreateTplFS, fmt.Sprintf("create/%s.tpl", tmpl))
    if err != nil {
        return err
    }
    var buffer bytes.Buffer
    err = t.Execute(&buffer, data)
    if err != nil {
        return err
    }
    result, err := imports.Process(fileName, buffer.Bytes(), nil)
    if err != nil {
        return err
    }
    return os.WriteFile(fileName, result, 0640)
}

func (g *Generator) checkStructName(name string) error {
    if name == "" {
        return nil
    }
    if !regexp.MustCompile(`^\w+$`).MatchString(name) {
        return fmt.Errorf("repo name cannot contains invalid character")
    }
    if name[0] < 'A' || name[0] > 'Z' {
        return fmt.Errorf("repo name must be initial capital")
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

// check if default tag needed
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

// return gorm default tag's value
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

// struct comment
func (b *StructMeta) StructComment() string {
    if b.TableName != "" {
        return fmt.Sprintf(`mapped from table <%s>`, b.TableName)
    }
    return `mapped from object`
}
