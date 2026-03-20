package agentsvc

import (
	"context"
	"encoding/json"

	"github.com/dragodui/my-deploy/internal/agent"
	agentpb "github.com/dragodui/my-deploy/internal/shared/proto/agentpb/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentGRPCServer struct {
	agentpb.UnimplementedAgentInternalServer
	Registry *AgentRegistry
	Repo     *AgentRepository
}

func (s *AgentGRPCServer) IsConnected(ctx context.Context, req *agentpb.IsConnectedRequest) (*agentpb.IsConnectedResponse, error) {
	_, ok := s.Registry.Get(req.AgentId)

	return &agentpb.IsConnectedResponse{IsConnected: ok}, nil
}

func (s *AgentGRPCServer) SendCommand(ctx context.Context, req *agentpb.SendCommandRequest) (*agentpb.SendCommandResponse, error) {
	ac, ok := s.Registry.Get(req.AgentId)
	if !ok {
		return nil, status.Error(codes.NotFound, "agent not connected")
	}
	res, err := ac.SendCommand(ctx, agent.Command{
		Type:    req.Cmd.Type,
		ID:      req.Cmd.Id,
		Payload: req.Cmd.Payload,
	})
	if err != nil {
		return nil, err
	}
	return &agentpb.SendCommandResponse{
		Id:          res.ID,
		Success:     res.Success,
		ContainerId: res.ContainerID,
		Error:       res.Error,
		Status:      res.Status,
	}, nil
}

func (s *AgentGRPCServer) GetAgent(ctx context.Context, req *agentpb.GetAgentRequest) (*agentpb.GetAgentResponse, error) {
	ag, err := s.Repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &agentpb.GetAgentResponse{
		Id:        ag.ID,
		UserId:    ag.UserID,
		Token:     ag.Token,
		Name:      ag.Name,
		MachineId: ag.MachineID,
	}, nil
}

func (s *AgentGRPCServer) StreamLogs(req *agentpb.StreamLogsRequest, stream agentpb.AgentInternal_StreamLogsServer) error {
	ac, ok := s.Registry.Get(req.AgentId)
	if !ok {
		return status.Error(codes.NotFound, "agent not connected")
	}

	cmdID := uuid.New().String()
	payload, _ := json.Marshal(agent.ContainerPayload{ContainerID: req.ContainerId})
	go ac.SendCommand(context.Background(), agent.Command{
		Type:    "logs",
		ID:      cmdID,
		Payload: payload,
	})

	logsChan := ac.SubscribeLogs(cmdID)
	defer ac.UnsubscribeLogs(cmdID)

	for chunk := range logsChan {
		if err := stream.Send(&agentpb.StreamLogsResponse{
			Data:  chunk.Data,
			Done:  chunk.Done,
			Error: chunk.Error,
		}); err != nil {
			return err
		}
		if chunk.Done {
			return nil
		}
	}

	return nil
}

func (s *AgentGRPCServer) GetProgress(ctx context.Context, req *agentpb.GetProgressRequest) (*agentpb.GetProgressResponse, error) {
	ac, ok := s.Registry.Get(req.AgentId)
	if !ok {
		return nil, status.Error(codes.NotFound, "agent not connected")
	}

	progress := ac.GetProgress(req.CmdId)
	return &agentpb.GetProgressResponse{
		Progress: progress,
	}, nil
}
