package transformer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/utils/misc"
)

type TransformerFeatureServiceConfig struct {
	Log                      logger.Logger
	PollInterval             time.Duration
	TransformerURL           string
	featuresRetryMaxAttempts int
}

type TransformerFeaturesService interface {
	GetSourceTransformerVersion() string
	GetTransformerFeatures() json.RawMessage
	Wait() chan struct{}
}

var defaultTransformerFeatures = `{
	"routerTransform": {
	  "MARKETO": true,
	  "HS": true
	}
  }`

func NewTransformerFeatureService(ctx context.Context, config TransformerFeatureServiceConfig) (TransformerFeaturesService, error) {
	if config.Log == nil {
		config.Log = logger.NewLogger().Child("transformer-features")
	}

	config.featuresRetryMaxAttempts = 10

	handler := &transformerHandler{
		logger:    config.Log,
		asyncInit: misc.NewAsyncInit(1),
		config:    config,
	}

	err := handler.init(ctx)

	return handler, err
}

func NewNoOpService() TransformerFeaturesService {
	return &noopService{}
}

type noopService struct{}

func (*noopService) GetSourceTransformerVersion() string {
	return "V0"
}

func (*noopService) Wait() chan struct{} {
	dummyChan := make(chan struct{})
	close(dummyChan)
	return dummyChan
}

func (*noopService) GetTransformerFeatures() json.RawMessage {
	return json.RawMessage(defaultTransformerFeatures)
}
