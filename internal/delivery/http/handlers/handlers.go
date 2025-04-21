package handlers

import "github.com/fajaragst/zoom-to-s3/internal/service"

type Handlers struct {
	Services *service.Services
	Record   *RecordHandlers
}

func NewHandlers(services *service.Services) *Handlers {
	return &Handlers{
		Services: services,
		Record:   NewRecordHandlres(services.RecordService),
	}
}
