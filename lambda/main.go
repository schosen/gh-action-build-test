package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
)

const (
	awsRegionUsEast1 string = "us-east-1"
	awsRegionUsWest2 string = "us-west-2"

	kinesisStreamName string = "faas-failover"
)

type MessageRecord struct {
	Type         string
	Message      interface{}
	Timestamp    int64
	SourceRegion string
}

var (
	kinesisClient kinesisiface.KinesisAPI
	streamName    string
)

func main() {

	streamName = os.Getenv("STREAM_NAME")
	if streamName == "" {
		streamName = kinesisStreamName
	}

	targetRegion := os.Getenv("AWS_REGION")
	if strings.ToLower(targetRegion) == awsRegionUsWest2 {
		targetRegion = awsRegionUsEast1
	} else {
		targetRegion = awsRegionUsWest2
	}
	log.Printf("Kinesis target data stream '%s' and region '%s'", streamName, targetRegion)

	awsSession, err := session.NewSession(&aws.Config{})
	if err != nil {
		log.Printf("Error creating session, err: %v", err)
		return
	}
	kinesisClient = kinesis.New(awsSession, &aws.Config{
		Region: aws.String(targetRegion),
	})

	runtime.Start(handleRequest)
}

func handleRequest(ctx context.Context, event events.KinesisEvent) (events.KinesisBatchItemFailure, error) {

	log.Printf("Starting process of %d event records", len(event.Records))

	// // Print event
	// eventJson, _ := json.MarshalIndent(event, "", "  ")
	// log.Printf("Event JSON: %s", eventJson)
	var batchItemFailures events.KinesisBatchItemFailure

	for _, eventRecord := range event.Records {
		if eventRecord.EventSource == "aws:kinesis" {

			batchItemFailures.ItemIdentifier = eventRecord.Kinesis.SequenceNumber

			msgDataRecord := MessageRecord{}
			err := json.Unmarshal(eventRecord.Kinesis.Data, &msgDataRecord)
			if err != nil {
				log.Printf("Failed to unmarshal message with sequence %s, err: %v", eventRecord.Kinesis.SequenceNumber, err)
				return batchItemFailures, err
			}

			log.Printf("Message received. Type: %s and Timestamp: %d", msgDataRecord.Type, msgDataRecord.Timestamp)

			err = SendMessage(ctx, kinesisClient, streamName, eventRecord.Kinesis)
			if err != nil {
				log.Printf("Failed sending a message with sequence %s, err: %v", eventRecord.Kinesis.SequenceNumber, err)
				return batchItemFailures, err
			}
		}
	}

	log.Printf("Done processing all records for this event, no errors detected")
	batchItemFailures.ItemIdentifier = ""

	return batchItemFailures, nil
}

func SendMessage(ctx context.Context, svc kinesisiface.KinesisAPI, streamName string, msgRecord events.KinesisRecord) error {

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	recOutput, err := svc.PutRecordWithContext(reqCtx, &kinesis.PutRecordInput{
		PartitionKey: aws.String(msgRecord.PartitionKey),
		Data:         msgRecord.Data,
		StreamName:   aws.String(streamName),
	})

	if err != nil {
		return err
	}

	log.Printf("Message forwarded successfully. ShardId: %s, SequenceNumber: %s", aws.StringValue(recOutput.ShardId), aws.StringValue(recOutput.SequenceNumber))

	return nil

}
