package healthcheck

import (
	"github.com/Peripli/service-manager/storage"
	"time"
)

type platformsIndicator struct {
	repository storage.Repository
}

// Name returns the name of the storage component
func (i *platformsIndicator) Name() string {
	return "platforms"
}

func (i *platformsIndicator) Interval() time.Duration {
	return 60 * time.Second
}

func (i *platformsIndicator) FailuresTreshold() int64 {
	return 1
}

func (i *platformsIndicator) Fatal() bool {
	return false
}

func (i *platformsIndicator) Status() (interface{}, error) {
	return nil, nil
}

//func CreatePlatformsHealthIndicators()
