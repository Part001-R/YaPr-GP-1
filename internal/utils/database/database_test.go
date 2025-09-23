package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_WorkDir(t *testing.T) {

	// Подготовка
	tempDir := t.TempDir()

	projectDir := filepath.Join(tempDir, nameHeadProject)

	err := os.MkdirAll(projectDir, 0755)
	require.NoErrorf(t, err, "неожиданная ошибка os.MkdirAll: <%v>", err)

	err = os.Chdir(projectDir)
	require.NoErrorf(t, err, "неожиданная ошибка os.Chdir: <%v>", err)

	// Тест
	result, err := workDir()
	require.NoErrorf(t, err, "неожиданная ошибка workDir: <%v>", err)

	want := nameHeadProject
	if !strings.HasPrefix(result, want) {
		t.Errorf("ожидалось, что результат будет начинаться с %q, но получено %q", want, result)
	}
}
