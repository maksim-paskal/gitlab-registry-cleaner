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
package metrics

import (
	"flag"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	log "github.com/sirupsen/logrus"
)

var (
	metricsJob     = flag.String("metrics.job", "gitlab_registry_cleaner", "")
	pushGateWayURL = flag.String("metrics.pushgateway", "", "URL to pushgateway http://localhost:9091")
)

const namespace = "gitlab_registry_cleaner"

var CompletionTime = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: namespace,
	Name:      "last_completion_timestamp_seconds",
	Help:      "The timestamp of the last successful completion.",
})

var TagsDeleted = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "tags_deleted_total",
	Help:      "Total deleted tags",
})

var TagsWarnings = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: namespace,
	Name:      "tags_warnings_total",
	Help:      "Total tags with warning",
})

func Push() error {
	if len(*pushGateWayURL) == 0 {
		return nil
	}

	log.Infof("send metrics to %s", *pushGateWayURL)

	if err := push.New(*pushGateWayURL, *metricsJob).
		Collector(CompletionTime).
		Collector(TagsDeleted).
		Collector(TagsWarnings).
		Push(); err != nil {
		return errors.Wrap(err, "can not send metrics")
	}

	return nil
}
