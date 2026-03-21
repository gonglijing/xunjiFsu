package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type DataAPI struct {
	service *service.DataService
}

func NewDataAPI(dataService *service.DataService) *DataAPI {
	return &DataAPI{service: dataService}
}
