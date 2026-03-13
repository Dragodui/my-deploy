package daemon

import (
	"os"
	"os/exec"
	"strconv"
)

func StartAgent(binaryPath string) error {
	if isRunning, _ := IsRunning(); isRunning {
		return nil
	}

	logFile, err := os.OpenFile(LogFilePath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	agentProcess := exec.Command(binaryPath)
	agentProcess.Stdout = logFile
	agentProcess.Stderr = logFile

	agentProcess.SysProcAttr = detachAttrs()

	if err := agentProcess.Start(); err != nil {
		return err
	}

	pidStr := strconv.Itoa(agentProcess.Process.Pid)
	if err := os.WriteFile(PidFilePath(), []byte(pidStr), 0644); err != nil {
		return err
	}

	return nil
}
