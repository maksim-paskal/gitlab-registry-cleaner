module github.com/maksim-paskal/gitlab-registry-cleaner

go 1.24.0

require (
	github.com/aws/aws-sdk-go v1.55.7
	github.com/heroku/docker-registry-client v0.0.0-20211012143308-9463674c8930
	github.com/maksim-paskal/logrus-hook-sentry v0.1.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/client_model v0.6.2
	github.com/sirupsen/logrus v1.9.3
	gitlab.com/gitlab-org/api/client-go v0.128.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/getsentry/sentry-go v0.32.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/prometheus/common v0.63.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	golang.org/x/oauth2 v0.29.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/heroku/docker-registry-client => github.com/maksim-paskal/docker-registry-client v0.0.0-20220428053414-1c2590a3d930
