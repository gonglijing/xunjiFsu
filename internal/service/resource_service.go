package service

import (
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type ResourceService struct {
	driverExecutor *driver.DriverExecutor
}

func NewResourceService(driverExecutor *driver.DriverExecutor) *ResourceService {
	return &ResourceService{driverExecutor: driverExecutor}
}

func (s *ResourceService) ListResources() ([]*models.Resource, error) {
	return database.ListResources()
}

func (s *ResourceService) LoadResource(id int64) (*models.Resource, error) {
	return database.GetResourceByID(id)
}

func (s *ResourceService) CreateResource(resource *models.Resource) (*models.Resource, error) {
	if resource == nil {
		return nil, nil
	}

	id, err := database.AddResource(resource)
	if err != nil {
		return nil, err
	}
	resource.ID = id
	return resource, nil
}

func (s *ResourceService) UpdateResource(resource *models.Resource) (*models.Resource, error) {
	if resource == nil {
		return nil, nil
	}

	if err := database.UpdateResource(resource); err != nil {
		return nil, err
	}
	if s.driverExecutor != nil {
		s.driverExecutor.RefreshResource(resource)
	}
	return resource, nil
}

func (s *ResourceService) DeleteResource(id int64) error {
	if err := database.DeleteResource(id); err != nil {
		return err
	}
	if s.driverExecutor != nil {
		s.driverExecutor.CloseResource(id)
	}
	return nil
}

func (s *ResourceService) ToggleResource(id int64) (*models.Resource, error) {
	resource, err := database.GetResourceByID(id)
	if err != nil {
		return nil, err
	}

	resource.Enabled = nextResourceEnabledState(resource.Enabled)
	if err := database.ToggleResource(resource.ID, resource.Enabled); err != nil {
		return nil, err
	}
	if s.driverExecutor != nil {
		s.driverExecutor.RefreshResource(resource)
	}
	return resource, nil
}

func nextResourceEnabledState(enabled int) int {
	if enabled == 0 {
		return 1
	}
	return 0
}
