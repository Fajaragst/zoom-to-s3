package service

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Deps struct {
	S3 *s3.Client
}

type Services struct {
	RecordService RecordService
}

func NewService(deps Deps) *Services {
	return &Services{
		RecordService: NewRecordService(deps.S3),
	}
}
