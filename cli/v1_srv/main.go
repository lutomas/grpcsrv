package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	pb "github.com/lutomas/grpcsrv/apis/grpcsrv/v1"
	"google.golang.org/grpc"
)

func main() {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	x := new(theSocialRobotServer)
	pb.RegisterTheSocialRobotServer(grpcServer, x)

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
		if s.currentStream == nil {
			s.currentStream = stream
			go s.simulate()
		}

		// TODO handle events from the client
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		log.Printf("Received event '%d'", event.Id)
		//// respond with a single command
		//// TODO eventually we'll decouple receiving events from sending commands
		//command := &pb.ServerEvent{
		//	Id:      event.Id,
		//	Actions: []*pb.Action{{Delay: 0, Action: &pb.Action_Say{Say: &pb.Say{Text: "Hello World"}}}},
		//}
		//stream.Send(command)
	}
}

func (s *theSocialRobotServer) simulate() {
	defer func() {
		s.currentStream = nil
	}()
	eventID := int32(0)
	for {

		command := &pb.ServerEvent{
			Id:      eventID,
			Actions: []*pb.Action{{Delay: 0, Action: &pb.Action_Date{Date: &pb.Date{Text: time.Now().Format(time.RFC3339Nano)}}}}}

		err := s.currentStream.Send(command)
		if err != nil {
			fmt.Printf("Failed to send command to server: %v\n", err)
			break
		}

		fmt.Printf("Sent command to server: %v\n", command)
		time.Sleep(time.Second)
	}
}
