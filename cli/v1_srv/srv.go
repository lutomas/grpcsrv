package main

import (
	"io"
	"log"

	pb "github.com/lutomas/grpcsrv/apis/grpcsrv/v1"
)

type theSocialRobotServer struct {
	pb.UnimplementedTheSocialRobotServer
	currentStream pb.TheSocialRobot_EventStreamServer
}

func (s *theSocialRobotServer) EventStream(stream pb.TheSocialRobot_EventStreamServer) error {
	for {
		s.currentStream = stream

		// TODO handle events from the client
		_, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		log.Printf("Received event, sending one command")
		// respond with a single command
		// TODO eventually we'll decouple receiving events from sending commands
		command := &pb.ServerEvent{
			Id:      1,
			Actions: []*pb.Action{{Delay: 0, Action: &pb.Action_Say{Say: &pb.Say{Text: "Hello World"}}}},
		}
		stream.Send(command)
	}
}
