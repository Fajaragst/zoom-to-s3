# Zoom to S3

A service that automatically uploads Zoom meeting recordings to Amazon S3 storage using Zoom's webhook notifications.

## Overview

This application listens for Zoom webhook events (recording.completed) when recordings are completed, then downloads the MP4 recording files and uploads them to your specified S3 bucket. 

## How It Works

1. Zoom sends a webhook notification when a recording is completed
2. The application validates the webhook signature
3. The application extracts the recording information and download URL
4. The MP4 recording is downloaded from Zoom
5. The recording is uploaded to your S3 bucket using multipart upload
6. The recording is stored with a path format: `{prefix}/{day}-{month}-{year}/{meeting-name}-{timestamp}.mp4`

## API Endpoints

### POST `/api/v1/`

This endpoint receives webhook notifications from Zoom when recordings are completed.

**Headers:**
- `X-Zm-Signature`: Signature for webhook verification
- `X-Zm-Request-Timestamp`: Timestamp of the request

**Request Body:**
The webhook payload from Zoom containing recording information.

**Response:**
- `200 OK`: Recording was successfully processed
- `401 Unauthorized`: Invalid webhook signature
- `500 Internal Server Error`: Error processing the recording

## Setup Instructions

### Prerequisites

- AWS Account with S3 access
- Zoom Developer Account with webhook capabilities
- Docker and Docker Compose (for containerized deployment)

### Configuration

1. Clone this repository
2. Create a `.env` file with the following variables:
   ```
   AWS_REGION=your-aws-region
   AWS_ACCESS_KEY_ID=your-access-key
   AWS_SECRET_ACCESS_KEY=your-secret-key
   S3_BUCKET=your-bucket-name
   S3_KEY_PREFIX=recordings
   AWS_ACCESS_KEY=your_aws_access_key
   AWS_SECRET_KEY=your_aws_secret_key
   ```