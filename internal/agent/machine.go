package agent

import "github.com/google/uuid"

func GenerateMachineID() string {
	return uuid.New().String()
}
