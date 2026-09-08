package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFieldTags(t *testing.T) {
	// 验证 Field.Tags() 仅生成 GORMTag，绝不生成 json tag
	f := &Field{
		Name:       "UserName",
		Type:       "string",
		GORMTag:    "column:user_name;type:varchar(64);not null",
		CommentTag: "// 用户名",
	}

	tags := f.Tags()
	if strings.Contains(tags, "json:") {
		t.Fatalf("expected no json tag in Tags(), got %q", tags)
	}
	expectedGORM := `gorm:"column:user_name;type:varchar(64);not null"`
	if tags != expectedGORM {
		t.Fatalf("expected %q, got %q", expectedGORM, tags)
	}
}

func TestGenerator_GenerateRepo_NoJSONTag(t *testing.T) {
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

	conf := SQLConfig{
		OutPath:     filepath.Join(tmpDir, "internal", "repository", "db"),
		PackageName: "db",
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
	// 验证绝对不包含 json tag，确保 DO 纯洁性
	if strings.Contains(contentStr, `json:`) {
		t.Fatalf("generated file should not contain json tag, content:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, `gorm:"column:id`) {
		t.Fatalf("generated file should contain gorm tag, content:\n%s", contentStr)
	}

	// 验证已移除反引号硬编码，兼容 PostgreSQL
	if strings.Contains(contentStr, "`id`") {
		t.Fatalf("generated file should not contain backtick-quoted column `id`, content:\n%s", contentStr)
	}
	// user_account 没有 deleted_at，不应生成 SoftDelete
	if strings.Contains(contentStr, "SoftDelete(") {
		t.Fatalf("table without deleted_at should not generate SoftDelete method, content:\n%s", contentStr)
	}
}

func TestHasSoftDeleteField(t *testing.T) {
	cases := []struct {
		fields []*Field
		want   bool
	}{
		{
			fields: []*Field{{ColumnName: "id"}, {ColumnName: "name"}},
			want:   false,
		},
		{
			fields: []*Field{{ColumnName: "id"}, {ColumnName: "deleted_at"}},
			want:   true,
		},
		{
			fields: []*Field{{ColumnName: "id"}, {ColumnName: "delete_at"}},
			want:   true,
		},
		{
			fields: []*Field{{ColumnName: "id"}, {Name: "DeletedAt"}},
			want:   true,
		},
	}
	for i, tc := range cases {
		got := hasSoftDeleteField(tc.fields)
		if got != tc.want {
			t.Errorf("case %d: expected %v, got %v", i, tc.want, got)
		}
	}
}
