package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractPKColumns(t *testing.T) {
	cases := []struct {
		name string
		line string
		want []string
	}{
		{"单列", "PRIMARY KEY (`id`)", []string{"id"}},
		{"单列无反引号", "PRIMARY KEY (id)", []string{"id"}},
		{"复合主键", "PRIMARY KEY (`user_id`,`role_id`)", []string{"user_id", "role_id"}},
		{"复合主键带空格", "PRIMARY KEY (`user_id`, `role_id`)", []string{"user_id", "role_id"}},
		{"前缀主键", "PRIMARY KEY (`name`(10))", []string{"name"}},
		{"USING BTREE", "PRIMARY KEY (`id`) USING BTREE", []string{"id"}},
		{"大小写混合", "primary key (`Id`)", []string{"Id"}},
		{"非主键行", "KEY `idx_name` (`name`)", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractPKColumns(c.line)
			if len(got) != len(c.want) {
				t.Fatalf("extractPKColumns(%q) = %v, want %v", c.line, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("extractPKColumns(%q) = %v, want %v", c.line, got, c.want)
				}
			}
		})
	}
}

func TestStripSQLComments(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		contains string // 结果里应当出现
		absent   string // 结果里不应出现
	}{
		{
			name:     "行注释 --",
			in:       "CREATE TABLE a ( -- this is a comment DROP TABLE x\n id int\n);",
			contains: "id int",
			absent:   "DROP TABLE x",
		},
		{
			name:     "行注释 #",
			in:       "CREATE TABLE a ( # comment CREATE TABLE fake\n id int\n);",
			contains: "id int",
			absent:   "fake",
		},
		{
			name:     "块注释",
			in:       "CREATE TABLE a (/* CREATE TABLE fake ( x int ) */ id int);",
			contains: "id int",
			absent:   "fake",
		},
		{
			name:     "mysqldump 条件注释",
			in:       "/*!40101 SET NAMES utf8 */;\nCREATE TABLE a (id int);",
			contains: "CREATE TABLE a",
			absent:   "40101",
		},
		{
			// 关键回归:注释内容出现 -- / # 时不能破坏引号配对
			name:     "字符串字面量内的注释符号必须保留",
			in:       "id int COMMENT '进度 -- 50% # 完成',",
			contains: "进度 -- 50% # 完成",
			absent:   "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripSQLComments(c.in)
			if c.contains != "" && !strings.Contains(got, c.contains) {
				t.Fatalf("stripSQLComments(%q) = %q, should contain %q", c.in, got, c.contains)
			}
			if c.absent != "" && strings.Contains(got, c.absent) {
				t.Fatalf("stripSQLComments(%q) = %q, should NOT contain %q", c.in, got, c.absent)
			}
		})
	}
}

func TestStripQuotedLiterals(t *testing.T) {
	in := "`status` tinyint NOT NULL DEFAULT '0' COMMENT '状态 AUTO_INCREMENT PRIMARY KEY'"
	got := stripQuotedLiterals(in)
	// 真实约束必须保留
	if !strings.Contains(strings.ToUpper(got), "NOT NULL") {
		t.Fatalf("real NOT NULL must survive: %q", got)
	}
	// 注释里的关键字必须被剥掉
	if strings.Contains(strings.ToUpper(got), "AUTO_INCREMENT") {
		t.Fatalf("AUTO_INCREMENT inside comment must be stripped: %q", got)
	}
	if strings.Contains(strings.ToUpper(got), "PRIMARY KEY") {
		t.Fatalf("PRIMARY KEY inside comment must be stripped: %q", got)
	}
}

// 注释里的关键字不得污染列属性 —— 中文注释里写 "不能为空 NOT NULL" 很常见。
func TestParseSQLFile_CommentKeywordsDoNotPolluteColumns(t *testing.T) {
	sql := "CREATE TABLE `demo` (\n" +
		"  `id` bigint NOT NULL AUTO_INCREMENT,\n" +
		"  `note` varchar(64) DEFAULT NULL COMMENT '备注 AUTO_INCREMENT PRIMARY KEY NOT NULL',\n" +
		"  PRIMARY KEY (`id`)\n" +
		") ENGINE=InnoDB;"

	dir := t.TempDir()
	path := filepath.Join(dir, "t.sql")
	if err := os.WriteFile(path, []byte(sql), 0o600); err != nil {
		t.Fatal(err)
	}

	metas, err := ParseSQLFile(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 table, got %d", len(metas))
	}
	m := metas[0]
	if !m.HasPrimaryKey || m.PrimaryKeyColumn != "id" {
		t.Fatalf("expected primary key 'id', got hasPK=%v col=%q", m.HasPrimaryKey, m.PrimaryKeyColumn)
	}
	for _, f := range m.Fields {
		if f.ColumnName == "note" {
			if f.IsPrimaryKey {
				t.Fatal("column 'note' must NOT be primary key (keyword came from its COMMENT)")
			}
			if strings.Contains(f.GORMTag, "autoIncrement") {
				t.Fatalf("column 'note' must NOT be autoIncrement, tag=%q", f.GORMTag)
			}
		}
	}
}

// 复合主键必须被识别(此前正则只匹配单列,主键会静默丢失)。
func TestParseSQLFile_CompositePrimaryKey(t *testing.T) {
	sql := "CREATE TABLE `user_role` (\n" +
		"  `user_id` bigint NOT NULL,\n" +
		"  `role_id` bigint NOT NULL,\n" +
		"  PRIMARY KEY (`user_id`,`role_id`)\n" +
		") ENGINE=InnoDB;"

	dir := t.TempDir()
	path := filepath.Join(dir, "t.sql")
	if err := os.WriteFile(path, []byte(sql), 0o600); err != nil {
		t.Fatal(err)
	}

	metas, err := ParseSQLFile(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 table, got %d", len(metas))
	}
	if !metas[0].HasPrimaryKey {
		t.Fatal("composite primary key must be detected")
	}
	var pkCount int
	for _, f := range metas[0].Fields {
		if f.IsPrimaryKey {
			pkCount++
		}
	}
	if pkCount != 2 {
		t.Fatalf("expected both columns marked primary key, got %d", pkCount)
	}
}

// 被注释掉的 CREATE TABLE 不应被解析成一张真实的表。
func TestParseSQLFile_CommentedOutTableIgnored(t *testing.T) {
	sql := "-- CREATE TABLE `ghost` (`id` int);\n" +
		"/* CREATE TABLE `ghost2` (`id` int); */\n" +
		"CREATE TABLE `real_tbl` (\n" +
		"  `id` bigint NOT NULL AUTO_INCREMENT,\n" +
		"  PRIMARY KEY (`id`)\n" +
		");"

	dir := t.TempDir()
	path := filepath.Join(dir, "t.sql")
	if err := os.WriteFile(path, []byte(sql), 0o600); err != nil {
		t.Fatal(err)
	}

	metas, err := ParseSQLFile(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 {
		var names []string
		for _, m := range metas {
			names = append(names, m.TableName)
		}
		t.Fatalf("expected only the real table, got %d: %v", len(metas), names)
	}
	if metas[0].TableName != "real_tbl" {
		t.Fatalf("expected real_tbl, got %q", metas[0].TableName)
	}
}
