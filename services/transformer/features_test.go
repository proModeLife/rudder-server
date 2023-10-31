package transformer

import (
	"context"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/stretchr/testify/require"
)

func TestNewTransformerFeatureService(t *testing.T) {
	ctx := context.Background()
	config := TransformerFeatureServiceConfig{
		Log:                      logger.NewLogger(),
		PollInterval:             time.Duration(1),
		featuresRetryMaxAttempts: 1,
	}
	t.Run("it should be able to intialize TransformerFeaturesService", func(t *testing.T) {
		handler, err := NewTransformerFeatureService(ctx, config)
		require.NoError(t, err)
		require.NotNil(t, handler)
		_, ok := handler.(TransformerFeaturesService)
		if !ok {
			t.Errorf("handler is not of type TransformerFeaturesService, got: %T", handler)
			require.Fail(t, "handler is not of type TransformerFeaturesService")
			return
		}
	})
}
