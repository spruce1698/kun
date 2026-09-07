package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFieldTags(t *testing.T) {
	// 默认无 jsonTag
	fWithoutJSON := &Field{
		Name:       "UserName",
		Type:       "string",
		GORMTag:    "column:user_name;type:varchar(64);not null",
		JSONTag:    "",
		CommentTag: "// 用户名",
	}

	tags := fWithoutJSON.Tags()
	if strings.Contains(tags, "json:") {
		t.Fatalf("expected no json tag in Tags(), got %q", tags)
	}
	expectedGORM := `gorm:"column:user_name;type:varchar(64);not null"`
	if tags != expectedGORM {
		t.Fatalf("expected %q, got %q", expectedGORM, tags)
	}

	// 启用 jsonTag
	fWithJSON := &Field{
		Name:       "UserName",
		Type:       "string",
		GORMTag:    "column:user_name;type:varchar(64);not null",
		JSONTag:    "userName",
		CommentTag: "// 用户名",
	}

	tagsWithJSON := fWithJSON.Tags()
	expectedWithJSON := `gorm:"column:user_name;type:varchar(64);not null" json:"userName"`
	if tagsWithJSON != expectedWithJSON {
		t.Fatalf("expected %q, got %q", expectedWithJSON, tagsWithJSON)
	}
}

func TestGenerator_GenerateRepo_DefaultNoJSONTag(t *testing.T) {
	sql := `CREATE TABLE user_account (
		id bigint NOT NULL AUTO_INCREMENT,
		username varchar(64) NOT NULL,
		email varchar(128) DEFAULT NULL,
		PRIMARY KEY (id)
	);`

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(sql), 0o600); err != nil {
		t.Fatal(err)
	}

	// 默认配置：FieldWithJSONTag 为 false
	conf := SQLConfig{
		OutPath:          filepath.Join(tmpDir, "internal", "repository", "db"),
		PackageName:      "db",
		FieldWithJSONTag: false,
	}

	metas, err := ParseSQLFile(sqlPath, &conf)
	if err != nil {
		t.Fatalf("ParseSQLFile failed: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 meta, got %d", len(metas))
	}

	gen := NewGenerator(conf)
	gen.AddRepoMeta(metas[0])

	genFile := filepath.Join(tmpDir, "internal", "repository", "db", metas[0].FileName+"_gen.go")
	if err := os.MkdirAll(filepath.Dir(genFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := gen.output("dbDefault", metas[0], genFile); err != nil {
		t.Fatalf("gen.output failed: %v", err)
	}

	content, err := os.ReadFile(genFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, `json:`) {
		t.Fatalf("generated file should not contain json tag, content:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `gorm:"column:id`) {
		t.Fatalf("generated file should contain gorm tag, content:\n%s", contentStr)
	}
}

func TestGenerator_GenerateRepo_WithJSONTag(t *testing.T) {
	sql := `CREATE TABLE user_account (
		id bigint NOT NULL AUTO_INCREMENT,
		username varchar(64) NOT NULL,
		email varchar(128) DEFAULT NULL,
		PRIMARY KEY (id)
	);`

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(sql), 0o600); err != nil {
		t.Fatal(err)
	}

	// 显式开启 FieldWithJSONTag
	conf := SQLConfig{
		OutPath:          filepath.Join(tmpDir, "internal", "repository", "db"),
		PackageName:      "db",
		FieldWithJSONTag: true,
	}

	metas, err := ParseSQLFile(sqlPath, &conf)
	if err != nil {
		t.Fatalf("ParseSQLFile failed: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("expected 1 meta, got %d", len(metas))
	}

	gen := NewGenerator(conf)
	gen.AddRepoMeta(metas[0])

	genFile := filepath.Join(tmpDir, "internal", "repository", "db", metas[0].FileName+"_gen.go")
	if err := os.MkdirAll(filepath.Dir(genFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := gen.output("dbDefault", metas[0], genFile); err != nil {
		t.Fatalf("gen.output failed: %v", err)
	}

	content, err := os.ReadFile(genFile)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `json:"id"`) {
		t.Fatalf("generated file should contain json:\"id\" tag, content:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `json:"username"`) {
		t.Fatalf("generated file should contain json:\"username\" tag, content:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `gorm:"column:id`) {
		t.Fatalf("generated file should contain gorm tag, content:\n%s", contentStr)
	}
}
