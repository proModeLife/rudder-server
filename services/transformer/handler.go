package transformer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/utils/misc"
)

type transformerHandler struct {
	log       logger.Logger
	asyncInit *misc.AsyncInit
	config    TransformerFeatureServiceConfig
	features  json.RawMessage
}

func (t *transformerHandler) GetSourceTransformerVersion() string {
	return "V0"
}

func (t *transformerHandler) Wait() chan struct{} {
	return t.asyncInit.Wait()
}

func (t *transformerHandler) init(ctx context.Context) error {
	go t.syncTransformerFeatureJson(ctx)
	return nil
}

func (t *transformerHandler) syncTransformerFeatureJson(ctx context.Context) {
	var initDone bool
	for {
		// TODO: Make a call to the transformer to get the features
		t.log.Info("Fetching transformer features")

		if t.features != nil && !initDone {
			initDone = true
			t.asyncInit.Done()
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(t.config.PollInterval):
		}
	}
}
