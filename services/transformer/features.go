package transformer

import (
	"context"
	"time"
	"github.com/rudderlabs/rudder-go-kit/config"
	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/utils/misc"
)

type TransformerFeatureServiceConfig struct {
	Log          logger.Logger
	PollInterval time.Duration
	transformerURL config.GetString("DEST_TRANSFORM_URL", "http://localhost:9090")
}

type TransformerFeaturesService interface {
	Wait() chan struct{}
}

// TOASK:
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
