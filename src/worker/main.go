package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type ChunkInfo struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type MessageBody struct {
	Bucket string    `json:"bucket"`
	Key    string    `json:"key"`
	Chunk  ChunkInfo `json:"chunk"`
}

type ProcessingResult struct {
	Bucket     string    `json:"bucket"`
	Key        string    `json:"key"`
	Chunk      ChunkInfo `json:"chunk"`
	WordCount  int       `json:"wordCount"`
	OutputPath string    `json:"outputPath"` // S3 path where results are stored
}

var (
	s3Client      *s3.S3
	downloader    *s3manager.Downloader
	uploader      *s3manager.Uploader
	outputBucket  string
	resultsPrefix string
)

func init() {
	session := session.Must(session.NewSession())
	s3Client = s3.New(session)
	downloader = s3manager.NewDownloader(session)
	uploader = s3manager.NewUploader(session)

	outputBucket = os.Getenv("OUTPUT_BUCKET")
	resultsPrefix = "results" // You can customize this if needed
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, message := range sqsEvent.Records {
		// Parse the message body
		var body MessageBody
		err := json.Unmarshal([]byte(message.Body), &body)
		if err != nil {
			log.Printf("Failed to unmarshal message body: %v", err)
			continue // Skip to the next message
		}

		fmt.Printf("Processing chunk: bucket=%s, key=%s, start=%d, end=%d\n", body.Bucket, body.Key, body.Chunk.Start, body.Chunk.End)

		// Download the chunk from S3
		file, err := os.CreateTemp("", "chunk-")
		if err != nil {
			log.Printf("Failed to create temp file: %v", err)
			continue
		}
		defer os.Remove(file.Name()) // Make sure to remove the temp file

		numBytes, err := downloader.Download(file, &s3.GetObjectInput{
			Bucket: aws.String(body.Bucket),
			Key:    aws.String(body.Key),
			Range:  aws.String(fmt.Sprintf("bytes=%d-%d", body.Chunk.Start, body.Chunk.End)),
		})
		if err != nil {
			log.Printf("Failed to download chunk: %v", err)
			continue
		}

		fmt.Printf("Downloaded chunk size: %d bytes\n", numBytes)

		// Process the chunk: count words
		wordCount := countWords(file.Name())

		// Prepare the result
		result := ProcessingResult{
			Bucket:    body.Bucket,
			Key:       body.Key,
			Chunk:     body.Chunk,
			WordCount: wordCount,
		}

		// Upload the result to the output bucket
		resultJSON, err := json.Marshal(result)
		if err != nil {
			log.Printf("Failed to marshal result to JSON: %v", err)
			continue
		}

		outputKey := fmt.Sprintf("%s/%s-%d-%d.json", resultsPrefix, body.Key, body.Chunk.Start, body.Chunk.End)
		_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(outputBucket),
			Key:    aws.String(outputKey),
			Body:   strings.NewReader(string(resultJSON)),
		})
		if err != nil {
			log.Printf("Failed to upload result to S3: %v", err)
			continue
		}

		result.OutputPath = fmt.Sprintf("s3://%s/%s", outputBucket, outputKey)
		fmt.Printf("Uploaded result to: %s\n", result.OutputPath)
	}

	return nil
}

func countWords(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open file for word counting: %v", err)
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)

	wordCount := 0
	for scanner.Scan() {
		wordCount++
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error during word counting: %v", err)
		return 0
	}

	return wordCount
}

func main() {
	lambda.Start(handler)
}
