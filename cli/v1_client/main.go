package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/lutomas/grpcsrv/apis/grpcsrv/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	serverAddr := "localhost:50051"
	conn, err := grpc.NewClient(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewTheSocialRobotClient(conn)

	runEventStream(client)
}

func runEventStream(client pb.TheSocialRobotClient) {
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	ctx := context.Background()
	stream, err := client.EventStream(ctx)
	if err != nil {
		log.Fatalf("client.EventStream failed: %v", err)
	}

	clientEvent := int32(0)
	startEvent := &pb.ClientEvent{Id: fmt.Sprintf("client:%d", clientEvent), Action: &pb.ClientEvent_Start{}}
	if err := stream.Send(startEvent); err != nil {
		log.Fatalf("client.EventStream: stream.Send(%v) failed: %v", startEvent, err)
	}

	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				log.Println("EOF")
				//close(waitc)
				//return
				time.Sleep(500 * time.Millisecond)
				continue
			}
			if err != nil {
				log.Fatalf("client.EventStream failed: %v", err)
			}
			log.Printf("Got message| id: %s; date: %s", in.Id, in.Date)
			clientEvent = clientEvent + 1
			callbackEvent := &pb.ClientEvent{Id: fmt.Sprintf("client:%d", clientEvent), Action: &pb.ClientEvent_Callback{
				Callback: &pb.Callback{Event: in},
			}}
			if err := stream.Send(callbackEvent); err != nil {
				log.Fatalf("client.EventStream: stream.Send(%v) failed: %v", callbackEvent, err)
			}
		}
	}()

	defer stream.CloseSend()
	<-waitc
}
