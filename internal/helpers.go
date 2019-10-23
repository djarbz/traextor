package internal

import (
	"fmt"
	"os"
	"time"
)

// Log prints a formatted log string
func Log(format string, vars ...interface{}) {
	fmt.Printf("["+time.Now().Format(time.RFC850)+"] "+format+"\n", vars...)
}

// CheckFileExists will verify that the config file exists
func CheckFileExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	} else if err != nil {
		Log("Error checking file %s: %v", file, err)
		return false
	} else {
		return true
	}
}

// WriteFile will write bytes to a file
func WriteFile(path string, filename string, content []byte) error {
	if err := CreateDir(path); err != nil {
		return err
	}

	f, err := os.Create(fmt.Sprintf("%s%c%s", path, os.PathSeparator, filename))
	if err != nil {
		return err
	}

	_, err = f.Write(content)
	if err != nil {
		return err
	}

	return f.Close()
}

// CreateDir will create a directory
func CreateDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// GetEnv will load the value of an environmental variable, or a default value if not set.
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
