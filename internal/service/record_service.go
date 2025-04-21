package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type RecordFiles struct {
	ID             string    `json:"id"`
	MeetingID      string    `json:"meeting_id"`
	RecordingStart time.Time `json:"recording_start"`
	RecordingEnd   time.Time `json:"recording_end"`
	FileType       string    `json:"file_type"`
	FileSize       int       `json:"file_size"`
	FileExtension  string    `json:"file_extension"`
	FileName       string    `json:"file_name"`
	DownloadURL    string    `json:"download_url"`
	Status         string    `json:"status"`
	RecordingType  string    `json:"recording_type"`
}

type RecordPayloadObject struct {
	ID             int64         `json:"id"`
	UUID           string        `json:"uuid"`
	HostID         string        `json:"host_id"`
	AccountID      string        `json:"account_id"`
	Topic          string        `json:"topic"`
	Type           int           `json:"type"`
	StartTime      time.Time     `json:"start_time"`
	Password       string        `json:"password"`
	Timezone       string        `json:"timezone"`
	Duration       int           `json:"duration"`
	ShareURL       string        `json:"share_url"`
	TotalSize      int           `json:"total_size"`
	RecordingCount int           `json:"recording_count"`
	RecordingFiles []RecordFiles `json:"recording_files"`
}

type RecordPayload struct {
	AccountID string `json:"account_id"`
	Object    RecordPayloadObject
}

type Record struct {
	Event         string        `json:"event"`
	EventTS       int64         `json:"event_ts"`
	Payload       RecordPayload `json:"payload"`
	DownloadToken string        `json:"download_token"`
}

type RecordService interface {
	UploadS3(c context.Context, input Record, s3Bucket string, s3KeyPrefix string) error
}

type recordService struct {
	S3 *s3.Client
}

func NewRecordService(s3 *s3.Client) *recordService {
	return &recordService{
		S3: s3,
	}
}

func (s *recordService) UploadS3(c context.Context, input Record, s3Bucket string, s3KeyPrefix string) error {
	if input.Event != "recording.completed" {
		log.Printf("Skipping non-recording.completed event: %s", input.Event)
		return nil
	}

	log.Println("Starting S3 upload process")

	// Find MP4 file
	mp4File, err := s.findMP4File(input.Payload.Object.RecordingFiles)
	if err != nil {
		log.Printf("Error finding MP4 file: %v", err)
		return err
	}
	log.Printf("Found MP4 file: %s, size: %d bytes", mp4File.FileName, mp4File.FileSize)

	// Generate key for S3
	mp4FileName := strings.ReplaceAll(input.Payload.Object.Topic, " ", "-")
	key := fmt.Sprintf("%s/%02d-%02d-%d/%s-%s.mp4", s3KeyPrefix,
		input.Payload.Object.StartTime.Day(),
		input.Payload.Object.StartTime.Month(),
		input.Payload.Object.StartTime.Year(),
		mp4FileName,
		fmt.Sprintf("%d", input.Payload.Object.StartTime.Unix()))

	log.Printf("Generated S3 key: %s", key)

	// Start multipart upload
	uploadID, err := s.startMultipartUpload(c, s3Bucket, key)
	if err != nil {
		log.Printf("Failed to start multipart upload: %v", err)
		return err
	}
	log.Printf("Started multipart upload with ID: %s", *uploadID)

	// Download file from Zoom
	log.Printf("Downloading file from Zoom URL: %s", mp4File.DownloadURL)
	resp, err := s.downloadFromZoom(mp4File.DownloadURL, input.DownloadToken)
	if err != nil {
		log.Printf("Failed to download from Zoom: %v", err)
		return err
	}
	defer resp.Body.Close()
	log.Println("Successfully connected to Zoom download URL")

	// Upload file in parts
	log.Println("Starting to upload file parts to S3")
	completedParts, err := s.uploadParts(c, resp.Body, s3Bucket, key, uploadID)
	if err != nil {
		log.Printf("Failed to upload parts: %v", err)
		return err
	}
	log.Printf("Successfully uploaded %d parts to S3", len(completedParts))

	// Complete the upload
	log.Println("Completing multipart upload")
	err = s.completeMultipartUpload(c, s3Bucket, key, uploadID, completedParts)
	if err != nil {
		log.Printf("Failed to complete multipart upload: %v", err)
		return err
	}

	log.Printf("Successfully uploaded recording to S3: s3://%s/%s", s3Bucket, key)
	return nil
}

func (s *recordService) findMP4File(files []RecordFiles) (*RecordFiles, error) {
	log.Printf("Searching for MP4 file among %d recording files", len(files))
	for i := range files {
		log.Printf("Checking file: %s, extension: %s", files[i].FileName, files[i].FileExtension)
		if files[i].FileExtension == "MP4" {
			return &files[i], nil
		}
	}
	return nil, fmt.Errorf("MP4 file not found")
}

func (s *recordService) startMultipartUpload(c context.Context, bucket, key string) (*string, error) {
	log.Printf("Starting multipart upload to bucket: %s, key: %s", bucket, key)
	createResp, err := s.S3.CreateMultipartUpload(c, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String("video/mp4"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start multipart upload: %w", err)
	}
	return createResp.UploadId, nil
}

func (s *recordService) downloadFromZoom(url, token string) (*http.Response, error) {
	log.Printf("Creating request to download from Zoom URL: %s", url)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	log.Println("Sending request to Zoom")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download source: %w", err)
	}

	log.Printf("Received response from Zoom with status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("bad status: %v", resp.Status)
	}

	return resp, nil
}

func (s *recordService) uploadParts(c context.Context, reader io.Reader, bucket, key string, uploadID *string) ([]types.CompletedPart, error) {
	const partSize = 5 * 1024 * 1024 // 5 MB
	log.Printf("Starting to upload parts with part size: %d bytes", partSize)

	buffer := make([]byte, partSize)
	partNumber := int32(1)
	var completedParts []types.CompletedPart

	for {
		log.Printf("Reading part %d from source", partNumber)
		n, err := io.ReadFull(reader, buffer)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			if n == 0 {
				log.Println("Reached end of file with 0 bytes read")
				break
			}
			log.Printf("Reached end of file with %d bytes read", n)
		} else if err != nil {
			return nil, fmt.Errorf("read error: %w", err)
		}

		log.Printf("Uploading part %d with %d bytes", partNumber, n)
		uploadPartResp, err := s.S3.UploadPart(c, &s3.UploadPartInput{
			Bucket:        aws.String(bucket),
			Key:           aws.String(key),
			PartNumber:    aws.Int32(partNumber),
			UploadId:      uploadID,
			Body:          io.NopCloser(bytes.NewReader(buffer[:n])),
			ContentLength: aws.Int64(int64(n)),
		})
		if err != nil {
			log.Printf("Failed to upload part %d: %v", partNumber, err)
			return nil, fmt.Errorf("upload part %d: %w", partNumber, err)
		}

		log.Printf("Successfully uploaded part %d with ETag: %s", partNumber, *uploadPartResp.ETag)
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       uploadPartResp.ETag,
			PartNumber: aws.Int32(partNumber),
		})

		partNumber++
		if err == io.EOF {
			break
		}
	}

	log.Printf("Completed uploading %d parts", len(completedParts))
	return completedParts, nil
}

func (s *recordService) completeMultipartUpload(c context.Context, bucket, key string, uploadID *string, parts []types.CompletedPart) error {
	log.Printf("Completing multipart upload for bucket: %s, key: %s with %d parts", bucket, key, len(parts))
	_, err := s.S3.CompleteMultipartUpload(c, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return fmt.Errorf("complete upload: %w", err)
	}
	log.Println("Multipart upload completed successfully")
	return nil
}
