/**
 * @Desc: 解析 SQL 文件中的 CREATE TABLE 语句，生成 StructMeta
 */

package kernel

import (
	"os"
	"regexp"
	"strings"

	"github.com/spruce1698/kun/pkg/fmt"
	"gorm.io/gorm/schema"
)

// 预编译正则表达式
var (
	reCreateTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + "`?" + `(\w+)` + "`?" + `\s*\(`)
	rePrimaryKey  = regexp.MustCompile(`(?i)PRIMARY\s+KEY\s*\(` + "`?" + `(\w+)` + "`?" + `\)`)
	reUniqueKey   = regexp.MustCompile(`(?i)UNIQUE\s+KEY\s+` + "`" + `(\w+)` + "`" + `\s*\(([^)]+)\)`)
	reComment     = regexp.MustCompile(`(?i)COMMENT\s+'((?:[^']|''|\\.)*)'`)
	reDefault     = regexp.MustCompile(`(?i)DEFAULT\s+('[^']*'|[\w.]+|NULL)`)
	reNoiseClause = regexp.MustCompile(`(?i)\b(CHARACTER\s+SET\s+\w+|COLLATE\s+\w+)\b`)
)

// columnDef 表示从 CREATE TABLE 语句中解析出的列定义
type columnDef struct {
	Name        string // 原始列名
	SQLType     string // 完整 SQL 类型，如 "decimal(10,2) unsigned"
	RawType     string // 基础类型，如 "decimal"
	Constraints string // 剩余约束
	IsUnsigned  bool   // 是否有 UNSIGNED 修饰
}

// uniqueIndexInfo 表示 UNIQUE KEY 索引信息
type uniqueIndexInfo struct {
	IndexName string
	Priority  int // 索引中的列顺序，从 1 开始
}

// ParseSQLFile 解析 .sql 文件，提取所有 CREATE TABLE 语句并构建 StructMeta
// conf 用于控制代码生成行为（FieldNullable、FieldCoverable、FieldSignable、FieldWithIndexTag、FieldWithTypeTag）
func ParseSQLFile(filePath string, conf *SQLConfig) ([]*StructMeta, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read sql file fail: %w", err)
	}

	// 处理 BOM 和 UTF-16 编码（Windows 系统可能将文件保存为 UTF-16 LE）
	content = convertToUTF8(content)

	sql := string(content)

	var metas []*StructMeta
	remaining := sql
	for {
		loc := reCreateTable.FindStringSubmatchIndex(remaining)
		if loc == nil {
			break
		}
		tableName := remaining[loc[2]:loc[3]]

		// loc[1]-1 是 '(' 的位置，找到匹配的 ')'
		parenStart := loc[1] - 1
		parenEnd := findMatchingParen(remaining, parenStart)
		if parenEnd < 0 {
			fmt.Warn("skip table <%s>: cannot find matching ')'", tableName)
			remaining = remaining[loc[1]:]
			continue
		}

		body := remaining[parenStart+1 : parenEnd]
		meta := parseTableBody(tableName, body, conf)
		if meta != nil {
			metas = append(metas, meta)
		}

		remaining = remaining[parenEnd+1:]
	}

	return metas, nil
}

// findMatchingParen 从 start 位置开始找到匹配的右括号，跳过字符串字面量
func findMatchingParen(s string, start int) int {
	depth := 0
	inString := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if ch == '\\' && i+1 < len(s) {
				i++ // 跳过反斜杠转义的字符
				continue
			}
			if ch == '\'' {
				// 检查转义的单引号 ''
				if i+1 < len(s) && s[i+1] == '\'' {
					i++ // 跳过转义的单引号
				} else {
					inString = false
				}
			}
			continue
		}
		if ch == '\'' {
			inString = true
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// parseTableBody 解析 CREATE TABLE 的列定义部分
func parseTableBody(tableName, body string, conf *SQLConfig) *StructMeta {
	structName := schema.NamingStrategy{}.SchemaName(tableName)
	fileName := toLowerCamel(structName)

	columns := splitByComma(body)

	// 第一遍：解析 UNIQUE KEY 索引信息
	uniqueIndexMap := parseUniqueIndexes(body)

	var fields []*Field
	primaryKeyType := "int64"

	for _, col := range columns {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}

		// 跳过约束/索引定义行
		if isConstraintLine(col) {
			// 提取 PRIMARY KEY 中的列名来标记主键
			if pkCol := extractPKColumn(col); pkCol != "" {
				for _, f := range fields {
					if strings.EqualFold(f.ColumnName, pkCol) {
						f.IsPrimaryKey = true
						primaryKeyType = f.Type
						if !strings.Contains(f.GORMTag, "primaryKey") {
							f.GORMTag += ";primaryKey"
						}
					}
				}
			}
			continue
		}

		cd := parseColumn(col)
		if cd == nil {
			continue
		}

		field := buildField(cd, uniqueIndexMap[cd.Name], conf)
		if field.IsPrimaryKey {
			primaryKeyType = field.Type
		}
		fields = append(fields, field)
	}

	var hasPrimaryKey bool
	var primaryKeyName string
	var primaryKeyColumn string

	// 1. First look for IsPrimaryKey = true
	for _, f := range fields {
		if f.IsPrimaryKey {
			hasPrimaryKey = true
			primaryKeyName = f.Name
			primaryKeyColumn = f.ColumnName
			primaryKeyType = f.Type
			break
		}
	}

	// 2. If not found, look for field Name == "Id" (case-insensitive column name "id")
	if !hasPrimaryKey {
		for _, f := range fields {
			if strings.ToLower(f.ColumnName) == "id" || f.Name == "Id" {
				hasPrimaryKey = true
				primaryKeyName = f.Name
				primaryKeyColumn = f.ColumnName
				primaryKeyType = f.Type
				break
			}
		}
	}

	if structName == "" {
		return nil
	}

	return &StructMeta{
		FileName:         fileName,
		InterfaceName:    fileName,
		StructName:       structName,
		TableName:        tableName,
		PackageName:      "db",
		PrimaryKeyType:   primaryKeyType,
		Fields:           fields,
		HasPrimaryKey:    hasPrimaryKey,
		PrimaryKeyName:   primaryKeyName,
		PrimaryKeyColumn: primaryKeyColumn,
	}
}

// splitByComma 按逗号切分，正确处理括号嵌套和字符串字面量
func splitByComma(s string) []string {
	var parts []string
	depth := 0
	inString := false
	start := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if ch == '\\' && i+1 < len(s) {
				i++ // 跳过反斜杠转义的字符
				continue
			}
			if ch == '\'' {
				// 检查转义的单引号 ''
				if i+1 < len(s) && s[i+1] == '\'' {
					i++ // 跳过转义的单引号
				} else {
					inString = false
				}
			}
			continue
		}
		if ch == '\'' {
			inString = true
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// isConstraintLine 判断是否是约束/索引行
func isConstraintLine(line string) bool {
	upper := strings.ToUpper(strings.TrimSpace(line))
	for _, prefix := range [...]string{"PRIMARY KEY", "KEY ", "INDEX ", "UNIQUE KEY", "UNIQUE INDEX", "CONSTRAINT", "FULLTEXT", "SPATIAL"} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

// extractPKColumn 从 PRIMARY KEY (`col`) 中提取列名
func extractPKColumn(line string) string {
	if m := rePrimaryKey.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	return ""
}

// parseUniqueIndexes 从 CREATE TABLE body 中解析 UNIQUE KEY 约束
func parseUniqueIndexes(body string) map[string][]uniqueIndexInfo {
	result := make(map[string][]uniqueIndexInfo)
	for _, m := range reUniqueKey.FindAllStringSubmatch(body, -1) {
		idxName := m[1]
		cols := strings.Split(m[2], ",")
		for i, col := range cols {
			col = strings.Trim(strings.TrimSpace(col), "`")
			result[col] = append(result[col], uniqueIndexInfo{
				IndexName: idxName,
				Priority:  i + 1,
			})
		}
	}
	return result
}

// parseColumn 手写解析器：列名 + 类型（含嵌套括号） + UNSIGNED? + 剩余约束
func parseColumn(line string) *columnDef {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// 1. 提取列名（可能被反引号包裹）
	colName, rest := extractQuotedName(line)
	if colName == "" {
		return nil
	}

	// 2. 提取类型：单词 + 可选括号参数（支持平衡括号）
	rawType, typeParams, rest := extractTypeWithParens(rest)
	if rawType == "" {
		return nil
	}

	fullType := strings.ToLower(rawType + typeParams)

	// 3. 检测 UNSIGNED / SIGNED
	isUnsigned := false
	rest = strings.TrimSpace(rest)
	if len(rest) >= 8 && strings.EqualFold(rest[:8], "UNSIGNED") {
		isUnsigned = true
		rest = strings.TrimSpace(rest[8:])
	} else if len(rest) >= 6 && strings.EqualFold(rest[:6], "SIGNED") {
		rest = strings.TrimSpace(rest[6:])
	}

	// 4. 移除噪声子句: CHARACTER SET xxx, COLLATE xxx
	rest = reNoiseClause.ReplaceAllString(rest, "")
	rest = compactSpaces(rest)

	// 5. 构建完整 SQL 类型（含 unsigned）
	sqlType := fullType
	if isUnsigned {
		sqlType += " unsigned"
	}

	return &columnDef{
		Name:        colName,
		SQLType:     strings.ToLower(sqlType),
		RawType:     strings.ToLower(rawType),
		Constraints: rest,
		IsUnsigned:  isUnsigned,
	}
}

// extractQuotedName 提取反引号包裹或裸单词的列名，返回 (name, remaining)
func extractQuotedName(s string) (string, string) {
	if strings.HasPrefix(s, "`") {
		end := strings.Index(s[1:], "`")
		if end < 0 {
			return "", s
		}
		return s[1 : end+1], strings.TrimSpace(s[end+2:])
	}
	// 裸单词
	parts := strings.SplitN(s, " ", 2)
	if len(parts) < 2 {
		return "", s
	}
	return parts[0], strings.TrimSpace(parts[1])
}

// extractTypeWithParens 提取类型单词 + 平衡括号参数，返回 (rawType, params, remaining)
func extractTypeWithParens(s string) (string, string, string) {
	s = strings.TrimSpace(s)
	spaceIdx := strings.Index(s, " ")
	parenIdx := strings.Index(s, "(")

	// 没有空格和括号 → 整个字符串就是类型
	if spaceIdx < 0 && parenIdx < 0 {
		return s, "", ""
	}

	// 类型名后紧跟括号: type(params)...
	if parenIdx >= 0 && (spaceIdx < 0 || parenIdx < spaceIdx) {
		rawType := s[:parenIdx]
		rest := s[parenIdx:]
		closeIdx := findMatchingParen(rest, 0)
		if closeIdx < 0 {
			return s, "", ""
		}
		params := normalizeTypeParams(rest[:closeIdx+1])
		remaining := strings.TrimSpace(rest[closeIdx+1:])
		return rawType, params, remaining
	}

	if spaceIdx > 0 {
		return s[:spaceIdx], "", strings.TrimSpace(s[spaceIdx:])
	}

	return s, "", ""
}

// normalizeTypeParams 去掉类型参数中逗号前后的空格: "(10, 2)" → "(10,2)"
func normalizeTypeParams(params string) string {
	if len(params) < 2 {
		return params
	}
	return "(" + strings.ReplaceAll(params[1:len(params)-1], " ", "") + ")"
}

// buildField 根据解析的列定义构建 kernel.Field
func buildField(cd *columnDef, uniqueIdxs []uniqueIndexInfo, conf *SQLConfig) *Field {
	// nil 安全兜底
	if conf == nil {
		conf = &SQLConfig{
			FieldNullable:     true,
			FieldCoverable:    false,
			FieldSignable:     false,
			FieldWithIndexTag: true,
			FieldWithTypeTag:  true,
		}
	}

	// 列名转 Go 字段名: snake_case → PascalCase, ID → Id
	fieldName := schema.NamingStrategy{}.SchemaName(cd.Name)
	fieldName = strings.ReplaceAll(fieldName, "ID", "Id")

	goType := GetSQLGoType(cd.RawType, cd.SQLType)

	// FieldSignable: 检测无符号整数，调整数据类型
	if conf.FieldSignable && cd.IsUnsigned && strings.HasPrefix(goType, "int") {
		goType = "u" + goType
	}

	isPK := false
	isAutoIncr := false
	isNullable := true // 默认 nullable，遇到 NOT NULL 才置 false
	defaultVal := ""
	comment := ""

	constraintsUpper := strings.ToUpper(cd.Constraints)

	// 提取 COMMENT
	if cre := reComment.FindStringSubmatch(cd.Constraints); cre != nil {
		comment = unescapeSQLString(cre[1])
	}

	// 提取 DEFAULT（支持字符串、数字、NULL）
	if dre := reDefault.FindStringSubmatch(cd.Constraints); dre != nil {
		if !strings.EqualFold(dre[1], "NULL") {
			defaultVal = strings.Trim(dre[1], "'")
		}
	}

	if strings.Contains(constraintsUpper, "PRIMARY KEY") || strings.Contains(constraintsUpper, "PRIMARY_KEY") {
		isPK = true
	}
	if strings.Contains(constraintsUpper, "AUTO_INCREMENT") {
		isAutoIncr = true
	}
	if strings.Contains(constraintsUpper, "NOT NULL") {
		isNullable = false
	}

	// 构建 GORM tag
	gormTag := "column:" + cd.Name
	// FieldWithTypeTag: 仅当启用时才添加 type tag
	if conf.FieldWithTypeTag {
		gormTag += ";type:" + cd.SQLType
	}
	if isAutoIncr {
		// AUTO_INCREMENT 列必然是主键，primaryKey 放在 autoIncrement 前面
		gormTag += ";primaryKey;autoIncrement:true"
		isPK = true
	} else if isPK {
		gormTag += ";primaryKey"
	}
	if !isNullable && !isPK && !isAutoIncr {
		gormTag += ";not null"
	}
	// FieldWithIndexTag: 仅当启用时才添加 UNIQUE KEY 索引信息
	if conf.FieldWithIndexTag {
		for _, idx := range uniqueIdxs {
			gormTag += fmt.Sprintf(";uniqueIndex:%s,priority:%d", idx.IndexName, idx.Priority)
		}
	}
	if defaultVal != "" && !isPK && !isZeroDefault(goType, defaultVal) {
		gormTag += ";default:" + defaultVal
	}

	// JSON tag: 小驼峰（传 fieldName 已含 ID→Id 修正）
	jsonTag := toLowerCamel(fieldName)

	// 注释标签：始终保留 //，有内容时追加
	commentTag := "//"
	if comment != "" {
		commentTag = "// " + comment
	}

	// 处理 deleted_at 特殊类型
	if cd.Name == "deleted_at" && goType == "time.Time" {
		goType = "gorm.DeletedAt"
	}

	// FieldCoverable: 当字段具有默认值时生成指针
	// FieldNullable: 当字段可为空时生成指针（主键和 deleted_at 除外）
	switch {
	case conf.FieldCoverable && defaultVal != "" && !isZeroDefault(goType, defaultVal) && !isPK && cd.Name != "deleted_at":
		goType = "*" + goType
	case conf.FieldNullable && isNullable && !isPK && cd.Name != "deleted_at":
		goType = "*" + goType
	}

	return &Field{
		Name:         fieldName,
		Type:         goType,
		GORMTag:      gormTag,
		JSONTag:      jsonTag,
		CommentTag:   commentTag,
		IsPrimaryKey: isPK,
		ColumnName:   cd.Name,
	}
}

// ────── 共用 helper ──────

// toLowerCamel PascalCase 首字母转小写，用于 fileName 和 JSON tag（与 DB 路径一致）
func toLowerCamel(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// compactSpaces 将连续空白压缩为单个空格
func compactSpaces(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

// unescapeSQLString 还原 SQL 字符串字面量中的转义：” → '，\' → '
func unescapeSQLString(s string) string {
	s = strings.ReplaceAll(s, "''", "'")
	s = strings.ReplaceAll(s, "\\'", "'")
	return s
}

// isZeroDefault 判断默认值是否是 Go 零值（无需写入 gorm default tag）
func isZeroDefault(goType, val string) bool {
	baseType := strings.TrimPrefix(goType, "*")
	switch baseType {
	case "int32", "int64", "uint32", "uint64", "float32", "float64":
		return val == "0"
	case "string":
		return val == ""
	case "bool":
		return val == "false"
	}
	return false
}

// ────── encoding ──────

// convertToUTF8 检测并转换 UTF-16/UTF-16LE-BOM 编码到 UTF-8
func convertToUTF8(data []byte) []byte {
	if len(data) >= 2 {
		// UTF-16 LE BOM: 0xFF 0xFE
		if data[0] == 0xFF && data[1] == 0xFE {
			return decodeUTF16LE(data[2:])
		}
		// UTF-16 BE BOM: 0xFE 0xFF
		if data[0] == 0xFE && data[1] == 0xFF {
			return decodeUTF16BE(data[2:])
		}
	}
	// 检测无 BOM 的 UTF-16 LE（启发式：前4字节中偶数位为非零ASCII，奇数位为0）
	if len(data) >= 4 && len(data)%2 == 0 {
		if data[0] != 0 && data[1] == 0 && data[2] != 0 && data[3] == 0 &&
			data[0] < 0x80 && data[2] < 0x80 {
			return decodeUTF16LE(data)
		}
	}
	return data
}

// decodeUTF16LE 将 UTF-16 LE 字节解码为 UTF-8
func decodeUTF16LE(data []byte) []byte {
	if len(data)%2 != 0 {
		return data
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		runes = append(runes, rune(data[i])|rune(data[i+1])<<8)
	}
	return []byte(string(runes))
}

// decodeUTF16BE 将 UTF-16 BE 字节解码为 UTF-8
func decodeUTF16BE(data []byte) []byte {
	if len(data)%2 != 0 {
		return data
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		runes = append(runes, rune(data[i])<<8|rune(data[i+1]))
	}
	return []byte(string(runes))
}
