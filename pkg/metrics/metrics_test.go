/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package metrics_test

import (
	"context"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/metrics"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var pushGateWayServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
	if err := checkPushGatewayResponse(req); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.WithError(err).Error()

	} else {
		writer.WriteHeader(http.StatusOK)
	}
}))

func checkPushGatewayResponse(req *http.Request) error {
	if req.Method != http.MethodPut {
		return errors.Errorf("request method %s not correct", req.Method)
	}

	if req.RequestURI != "/metrics/job/gitlab_registry_cleaner" {
		return errors.Errorf("request %s uri not correct", req.RequestURI)
	}

	defer req.Body.Close()

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read request body")
	}

	bodyString := string(b)

	if !strings.Contains(bodyString, "gitlab_registry_cleaner_tags_deleted_total") {
		return errors.Errorf("request body %s not correct", bodyString)
	}

	if !strings.Contains(bodyString, "gitlab_registry_cleaner_tags_deleted_total") {
		return errors.Errorf("request body %s not correct", bodyString)
	}

	return nil
}

func TestPush(t *testing.T) {
	if err := flag.Set("metrics.pushgateway", pushGateWayServer.URL); err != nil {
		t.Fatal(err)
	}

	t.Parallel()

	ctx := context.Background()

	if err := metrics.Push(ctx); err != nil {
		t.Fatal(err)
	}

	// fake pushgateway server should return error
	if err := flag.Set("metrics.pushgateway", "http://127.0.0.1:31212"); err != nil {
		t.Fatal(err)
	}

	if err := metrics.Push(ctx); err == nil {
		t.Fatal("push must return error")
	}

	// empty pushgateway server should return nil
	if err := flag.Set("metrics.pushgateway", ""); err != nil {
		t.Fatal(err)
	}

	if err := metrics.Push(ctx); err != nil {
		t.Fatal("push must return nil")
	}
}
