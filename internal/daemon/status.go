package daemon

import (
	"os"
	"strconv"
	"strings"
)

func IsRunning() (bool, int) {
	pidFile := PidFilePath()

	if pidFile == "" {
		return false, 0
	}

	pidFileInfo, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidFileInfo)))
	if err != nil {
		return false, 0
	}

	if isAlive := isProcessAlive(pid); !isAlive {
		return false, 0
	}

	return true, pid
}
