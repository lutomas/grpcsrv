package main

import (
	"fmt"
	"io"
	"log"
	"net"

	pb "github.com/lutomas/grpcsrv/apis/grpcsrv/v1"
	"google.golang.org/grpc"
)

func main() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	pb.RegisterTheSocialRobotServer(grpcServer, new(theSocialRobotServer))

	port := 50051 // we'll implement command line arguments leter
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer.Serve(lis)

}

type theSocialRobotServer struct {
	pb.UnimplementedTheSocialRobotServer
	currentStream pb.TheSocialRobot_EventStreamServer
}

func (s *theSocialRobotServer) EventStream(stream pb.TheSocialRobot_EventStreamServer) error {
	for {
		s.currentStream = stream

		// TODO handle events from the client
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		log.Printf("Received event '%d', sending one command", event.Id)
		// respond with a single command
		// TODO eventually we'll decouple receiving events from sending commands
		command := &pb.ServerEvent{
			Id:      event.Id,
			Actions: []*pb.Action{{Delay: 0, Action: &pb.Action_Say{Say: &pb.Say{Text: "Hello World"}}}},
		}
		stream.Send(command)
	}
}
