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
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/bucketcleaner"
	log "github.com/sirupsen/logrus"
)

var (
	bucket       = flag.String("bucket", os.Getenv("CLEANER_BUCKET"), "bucket to purge")
	folder       = flag.String("folder", os.Getenv("CLEANER_FOLDER"), "folder to purge")
	pattern      = flag.String("pattern", os.Getenv("CLEANER_PATTERN"), "pattern to purge")
	maxGroupSize = flag.Int("maxGroupSize", 10, "max group size") //nolint:gomnd
	dryRun       = flag.Bool("dryRun", false, "dry run")
)

func main() {
	flag.Parse()

	config := aws.Config{}

	if minioEndpoint := os.Getenv("MINIO_ENDPOINT"); minioEndpoint != "" {
		config = aws.Config{
			Endpoint:         aws.String(minioEndpoint),
			DisableSSL:       aws.Bool(false),
			S3ForcePathStyle: aws.Bool(true),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChanInterrupt := make(chan os.Signal, 1)
	signal.Notify(signalChanInterrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChanInterrupt
		log.Warn("Received an interrupt, stopping services...")
		cancel()
		<-signalChanInterrupt
		os.Exit(1)
	}()

	cleaner := bucketcleaner.NewBucketCleaner()

	cleaner.Pattern = *pattern
	cleaner.MaxSize = *maxGroupSize
	cleaner.DryRun = *dryRun

	if err := cleaner.Validate(); err != nil {
		log.WithError(err).Fatal("failed to validate")
	}

	if err := cleaner.Init(config); err != nil {
		log.WithError(err).Fatal("failed to init")
	}

	if err := cleaner.PurgeBucket(ctx, *bucket, *folder); err != nil {
		log.WithError(err).Fatal("failed to purge bucket")
	}
}
