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
	currentStream   pb.TheSocialRobot_EventStreamServer
	lastServerEvent *pb.ServerEvent
	callbackChan    chan bool
}

func (s *theSocialRobotServer) EventStream(stream pb.TheSocialRobot_EventStreamServer) error {
	for {
		// TODO handle events from the client
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch op := event.Action.(type) {
		case *pb.ClientEvent_Start:
			log.Printf("Received start '%s'", event.Id)
			if s.currentStream == nil {
				s.currentStream = stream
				go s.simulate()
			}
		case *pb.ClientEvent_Callback:
			// Verify if callback is same as lastServerEvent
			if s.lastServerEvent != nil && s.lastServerEvent.Id == op.Callback.Event.Id {
				// TODO release chanel
				s.callbackChan <- true
				s.lastServerEvent = nil
				log.Printf("OK: Received expected callback '%s' -'%s' - '%s'", event.Id, op.Callback.Event.Date, time.Now().Format(time.RFC3339Nano))
			} else {
				log.Printf("ERR: Received callbackChan '%s'", event.Id)
			}
		}
	}
}

func (s *theSocialRobotServer) simulate() {
	s.callbackChan = make(chan bool)
	defer func() {
		s.currentStream = nil
		close(s.callbackChan)
	}()
	eventID := int32(0)
	for {
		eventID = eventID + 1
		command := &pb.ServerEvent{
			Id:   fmt.Sprintf("srv:%d", eventID),
			Date: time.Now().Format(time.RFC3339Nano),
		}

		err := s.currentStream.Send(command)
		if err != nil {
			fmt.Printf("Failed to send command to server: %v\n", err)
			break
		}

		fmt.Printf("Simulated event: %v\n", command)
		s.lastServerEvent = command
		<-s.callbackChan
		time.Sleep(1 * time.Second)
	}
}
