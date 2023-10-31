package transformer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/utils/misc"
	"github.com/stretchr/testify/require"
)

func TestSyncTransformerFeatureJson(t *testing.T) {
	ctx := context.Background()
	config := TransformerFeatureServiceConfig{
		Log:                      logger.NewLogger(),
		PollInterval:             time.Duration(1),
		featuresRetryMaxAttempts: 1,
	}
	handler := &transformerHandler{
		logger:    logger.NewLogger(),
		asyncInit: misc.NewAsyncInit(1),
		config:    config,
	}
	t.Run("it should be able to intialise feature with default ones", func(t *testing.T) {
		handler.init(ctx) // initialising it
		features := handler.GetTransformerFeatures()
		sourceVersion := handler.GetSourceTransformerVersion()
		// since no call will be made to transformer so all values would be default
		require.Equal(t, features, json.RawMessage(defaultTransformerFeatures))
		require.Equal(t, sourceVersion, "v0")
	})
}
