package main

import (
	"context"
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

	event := &pb.ClientEvent{Id: int32(0)}
	if err := stream.Send(event); err != nil {
		log.Fatalf("client.EventStream: stream.Send(%v) failed: %v", event, err)
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
			log.Printf("Got message %d", in.Id)
			for _, action := range in.Actions {
				switch op := action.Action.(type) {
				case *pb.Action_Say:
					log.Printf("delay %d, say %s", action.Delay, op.Say.Text)
				case *pb.Action_Date:
					log.Printf("delay %d, date %s", action.Delay, op.Date.Text)
				}
			}
		}
	}()
	//event := &pb.ClientEvent{Id: 2}
	//if err := stream.Send(event); err != nil {
	//	log.Fatalf("client.EventStream: stream.Send(%v) failed: %v", event, err)
	//}
	defer stream.CloseSend()
	<-waitc
}
