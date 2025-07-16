package gateway

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func TestWriteSessionFile(t *testing.T) {
	// .envファイルを読み込む
	err := godotenv.Load()
	if err != nil {
		t.Errorf("Error loading .env file: %v", err)
	}

	jsonString := os.Getenv("SESSION_JSON")
	if jsonString == "" {
		t.Errorf("'SESSION_JSON' is not set")
	}

	dirPath := ".td"
	filePath := filepath.Join(dirPath, "session.json")

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if err != nil {
		if os.IsExist(err) {
		} else {
			t.Errorf("failed to open file %v", err)
		}
	} else {
		_, err = file.WriteString(jsonString)
		if err != nil {
			t.Errorf("failed to write file %v", err)
		}
	}
	file.Close()

	t.Log("complete write file")
}
