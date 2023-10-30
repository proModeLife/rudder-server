package transformer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rudderlabs/rudder-go-kit/config"
	"github.com/rudderlabs/rudder-go-kit/logger"
	"github.com/rudderlabs/rudder-server/utils/httputil"
	"github.com/rudderlabs/rudder-server/utils/misc"
	"github.com/tidwall/gjson"
)

type transformerHandler struct {
	logger    logger.Logger
	asyncInit *misc.AsyncInit
	config    TransformerFeatureServiceConfig
	features  json.RawMessage
}

func (t *transformerHandler) GetSourceTransformerVersion() string {
	if gjson.Get(string(t.GetTransformerFeatures()), "supportSourceTransformV1").Bool() {
		return "v1"
	}

	return "v0"
}

func (t *transformerHandler) GetTransformerFeatures() json.RawMessage {
	if t.features == nil {
		return json.RawMessage(defaultTransformerFeatures)
	}

	return t.features
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
		t.logger.Infof("Fetching transformer features from %s", t.config.TransformerURL)
		for {
			for i := 0; i < t.config.featuresRetryMaxAttempts; i++ {

				if ctx.Err() != nil {
					return
				}

				retry := t.makeFeaturesFetchCall()
				if retry {
					t.logger.Infof("Fetched transformer features from %s (retry: %v)", t.config.TransformerURL, retry)
				}
				if retry {
					select {
					case <-ctx.Done():
						return
					case <-time.After(200 * time.Millisecond):
						continue
					}
				}
				break
			}

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
}

func (t *transformerHandler) makeFeaturesFetchCall() bool {
	url := t.config.TransformerURL + "/features"
	req, err := http.NewRequest("GET", url, bytes.NewReader([]byte{}))
	if err != nil {
		t.logger.Error("error creating request - %s", err)
		return true
	}
	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: config.GetDuration("HttpClient.processor.timeout", 30, time.Second)}
	res, err := client.Do(req)
	if err != nil {
		t.logger.Error("error sending request - %s", err)
		return true
	}

	defer func() { httputil.CloseResponse(res) }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return true
	}

	if res.StatusCode == 200 {
		t.features = body
	} else if res.StatusCode == 404 {
		t.features = json.RawMessage(defaultTransformerFeatures)
	}

	return false
}
