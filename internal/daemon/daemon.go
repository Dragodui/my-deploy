package daemon

import (
	"os"
	"path/filepath"
)

func getFile(ext string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mydeploy", "agent."+ext)
}

func PidFilePath() string {
	return getFile("pid")
}

func LogFilePath() string {
	return getFile("log")
}
