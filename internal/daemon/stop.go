package daemon

import "os"

func Stop() error {
	isRunning, pid := IsRunning()
	if isRunning {
		cmd, err := os.FindProcess(pid)
		if err != nil {
			return err
		}

		if err := cmd.Kill(); err != nil {
			return err
		}

		if err := os.Remove(PidFilePath()); err != nil {
			return err
		}
	}

	return nil
}
