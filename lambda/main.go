package lambda

import (
	"log"
	"os"
	"strings"
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
	streamName string
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

}

//comment
