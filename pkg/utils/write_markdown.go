package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func WriteMarkdown(filePath, fileName, content string) error {
	if err := os.MkdirAll(filePath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", filePath, err)
	}

	filePath = filepath.Join(filePath, fileName)
	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %v", filePath, err)
	}
	log.Printf("written to: %s", filePath)
	return nil
}
