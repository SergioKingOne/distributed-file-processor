package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
)

// dependencies
var (
	s3Client  *s3.S3
	snsClient *sns.SNS
	chunkSize int64
)

func init() {
	session := session.Must(session.NewSession())
	s3Client = s3.New(session)
	snsClient = sns.New(session)

	chunkSizeEnv := os.Getenv("CHUNK_SIZE")
	if chunkSizeEnv == "" {
		chunkSize = int64(10 * 1024 * 1024)
	} else {
		parsedChunkSize, err := strconv.ParseInt(chunkSizeEnv, 10, 64)
		if err != nil {
			panic(err)
		}
		chunkSize = parsedChunkSize
	}
}

type ChunkInfo struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	snsTopicArn := os.Getenv("SNS_TOPIC_ARN")

	for _, record := range s3Event.Records {
		s3ObjectKey := record.S3.Object.Key
		s3BucketName := record.S3.Bucket.Name

		// Get the object size
		headObjectOutput, err := s3Client.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(s3BucketName),
			Key:    aws.String(s3ObjectKey),
		})
		if err != nil {
			return fmt.Errorf("failed to get object metadata: %v", err)
		}

		objectSize := *headObjectOutput.ContentLength

		// Calculate chunk boundaries
		var chunks []ChunkInfo
		for start := int64(0); start < objectSize; start += chunkSize {
			end := start + chunkSize - 1
			if end >= objectSize {
				end = objectSize - 1
			}
			chunks = append(chunks, ChunkInfo{Start: start, End: end})
		}

		fmt.Printf("Object: s3://%s/%s, Size: %d, Chunks: %d\n", s3BucketName, s3ObjectKey, objectSize, len(chunks))

		// Publish a message to SNS for each chunk
		for i, chunk := range chunks {
			message := fmt.Sprintf(`{"bucket": "%s", "key": "%s", "chunk": {"start": %d, "end": %d}}`, s3BucketName, s3ObjectKey, chunk.Start, chunk.End)

			_, err := snsClient.Publish(&sns.PublishInput{
				Message:  &message,
				TopicArn: &snsTopicArn,
				MessageAttributes: map[string]*sns.MessageAttributeValue{
					"ChunkNumber": {
						DataType:    aws.String("Number"),
						StringValue: aws.String(strconv.Itoa(i)),
					},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to publish message to SNS: %v", err)
			}

			fmt.Printf("Published chunk %d/%d to SNS\n", i+1, len(chunks))
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}