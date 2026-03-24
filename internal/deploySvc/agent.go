package deploysvc

import (
	"log"

	agentpb "github.com/dragodui/my-deploy/internal/shared/proto/agentpb/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewAgentClient(agentURL string) agentpb.AgentInternalClient {
	conn, err := grpc.NewClient(agentURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to agent service: %v", err)
	}
	return agentpb.NewAgentInternalClient(conn)
}
