package util

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func SaveJson(path string, data interface{}) error {
	// if path doesn't exist create it
	if _, err := os.Stat(path); os.IsNotExist(err) {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = file.Write(bs)
	return err
}
