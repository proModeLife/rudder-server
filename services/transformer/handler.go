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
)

type transformerHandler struct {
	log       logger.Logger
	asyncInit *misc.AsyncInit
	config    TransformerFeatureServiceConfig
	features  json.RawMessage
}

var defaultTransformerFeatures = `{
	"routerTransform": {
	  "MARKETO": true,
	  "HS": true
	}
  }`

func (t *transformerHandler) GetSourceTransformerVersion() string {
	if(t.features.supportSourceTransformV1 === true){
		return "v1"
	}
	return "v0"
}

func (t *transformerHandler) GetTranformerRouterFeatures() string {
	return t.features.routerTransform ||t.features = json.RawMessage(defaultTransformerFeatures);
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
		t.log.Info("Fetching transformer features")
		t.makeFeaturesFetchCall()
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
func (t *transformerHandler) makeFeaturesFetchCall() {
	url := t.config.transformerURL + "/features"
	req, err := http.NewRequest("GET", url, bytes.NewReader([]byte{}))
	if err != nil {
		t.log.Error("error creating request - %s", err)
		return
	}
	tr := &http.Transport{}
	client := &http.Client{Transport: tr, Timeout: config.GetDuration("HttpClient.processor.timeout", 30, time.Second)}
	res, err := client.Do(req)
	if err != nil {
		t.log.Error("error sending request - %s", err)
		return
	}

	defer func() { httputil.CloseResponse(res) }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	if res.StatusCode == 200 {
		t.features = body
	}
}
