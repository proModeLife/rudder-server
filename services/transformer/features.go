package transformer

import (
	"context"
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/utils/misc"
)

type TransformerFeatureServiceConfig struct {
	Log          logger.Logger
	PollInterval time.Duration
}

type TransformerFeaturesService interface {
	GetSourceTransformerVersion() string
	Wait() chan struct{}
}

func NewTransformerFeatureService(ctx context.Context, config TransformerFeatureServiceConfig) (TransformerFeaturesService, error) {
	if config.Log == nil {
		config.Log = logger.NewLogger().Child("transformer-features")
	}

	handler := &transformerHandler{
		log:       config.Log,
		asyncInit: misc.NewAsyncInit(1),
		config:    config,
	}

	err := handler.init(ctx)

	return handler, err
}

/* func NewNoOpService() TransformerFeaturesService {
	return &noopService{}
}

type noopService struct{}

func (*noopService) GetSourceTransformerVersion() string {
	return "V0"
} */
