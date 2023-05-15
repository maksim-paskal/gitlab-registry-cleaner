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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	"github.com/paskal-maksim/gitlab-registry-cleaner/internal"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/api"
	log "github.com/sirupsen/logrus"
)

var (
	logLevelConfig = flag.String("log.level", "INFO", "")
	logPretty      = flag.Bool("log.pretty", false, "")
	version        = flag.Bool("version", false, "")
)

const gracefulShutdownTimeout = 5 * time.Second

func main() {
	flag.Parse()

	if *version {
		fmt.Println(api.GetVersion()) //nolint:forbidigo
		os.Exit(0)
	}

	logLevel, err := log.ParseLevel(*logLevelConfig)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetLevel(logLevel)
	log.SetReportCaller(true)

	if *logPretty {
		log.SetFormatter(&log.TextFormatter{})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}

	hookSentry, err := logrushooksentry.NewHook(logrushooksentry.Options{
		Release: api.GetVersion(),
	})
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.AddHook(hookSentry)
	defer hookSentry.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.RegisterExitHandler(func() {
		cancel()
		time.Sleep(gracefulShutdownTimeout)
	})

	internal.Init()

	if err := internal.Run(ctx); err != nil {
		log.WithError(err).Fatal()
	}

	<-ctx.Done()
	time.Sleep(gracefulShutdownTimeout)
}
