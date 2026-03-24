package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func FindAgentBinary() (string, error) {
	binaryName := ""
	if runtime.GOOS == "windows" {
		binaryName = "mydeploy-agent.exe"
	} else {
		binaryName = "mydeploy-agent"
	}
	path, err := exec.LookPath(binaryName)
	if err == nil {
		return path, nil
	}

	exePath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), binaryName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", err
}
