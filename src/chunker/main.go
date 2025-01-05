package main

    import (
        "context"
        "fmt"
        "github.com/aws/aws-lambda-go/events"
        "github.com/aws/aws-lambda-go/lambda"
        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/s3"
    )

    // dependencies
    var (
        s3Client *s3.S3
    )

    func init() {
        session := session.Must(session.NewSession())
        s3Client = s3.New(session)
    }

    func handler(ctx context.Context, s3Event events.S3Event) error {
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

            // Calculate chunk size and number of chunks (implement your logic here)
            chunkSize := int64(10 * 1024 * 1024) // Example: 10 MB chunks
            numChunks := (objectSize + chunkSize - 1) / chunkSize

            fmt.Printf("Object: s3://%s/%s, Size: %d, Chunks: %d\n", s3BucketName, s3ObjectKey, objectSize, numChunks)

            // TODO: Publish messages to SNS for each chunk
        }

        return nil
    }

    func main() {
        lambda.Start(handler)
    }