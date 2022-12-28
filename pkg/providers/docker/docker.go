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
package docker

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	registryWait     = flag.Bool("registry-wait", false, "")
	registryURL      = flag.String("registry.url", utils.GetEnv("REGISTRY_URL", "http://127.0.0.1:5000"), "format https://registry.com") //nolint:lll
	registryLogin    = flag.String("registry.username", os.Getenv("REGISTRY_USERNAME"), "")
	registryPassword = flag.String("registry.password", os.Getenv("REGISTRY_PASSWORD"), "")
)

const (
	pingTimeout  = 5 * time.Second
	waitInterval = 3 * time.Second
)

type Provider struct {
	dryRun bool
	hub    *registry.Registry
}

func (p *Provider) pingRegistry(ctx context.Context) error {
	client := &http.Client{
		Timeout: pingTimeout,
	}

	url := fmt.Sprintf("%s/v2/", utils.FormatURL(*registryURL))

	log.Infof("waiting for registry %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "error making request")
	}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		log.WithError(err).Debug()

		return errors.Wrap(err, "error from ping")
	}

	return nil
}

// Create new client.
func (p *Provider) Init(dryRun bool) error {
	p.dryRun = dryRun

	ctx := context.Background()

	if *registryWait {
		for p.pingRegistry(ctx) != nil {
			time.Sleep(waitInterval)
		}
	}

	var err error

	p.hub, err = registry.New(*registryURL, *registryLogin, *registryPassword)
	if err != nil {
		return errors.Wrap(err, "can not connect to registry")
	}

	if log.GetLevel() < log.DebugLevel {
		p.hub.Logf = registry.Quiet
	}

	return nil
}

// List repositories.
func (p *Provider) Repositories(filter string) ([]string, error) {
	repos, err := p.hub.Repositories()
	if err != nil {
		return nil, errors.Wrap(err, "can not get repositories")
	}

	if len(filter) > 0 {
		return utils.FilterStrings(repos, filter), nil
	}

	return repos, nil
}

// List tags.
func (p *Provider) Tags(repository string) ([]string, error) {
	tags, err := p.hub.Tags(repository)

	return tags, errors.Wrap(err, "can not get tags")
}

// Delete tag.
func (p *Provider) DeleteTag(deleteTag types.DeleteTagInput) error {
	digest, err := p.hub.ManifestDigest(deleteTag.Repository, deleteTag.Tag)
	if err != nil {
		return errors.Wrap(err, "can not get digest")
	}

	if p.dryRun {
		log.Warn("nothing to do, dry run")

		return nil
	}

	err = p.hub.DeleteManifest(deleteTag.Repository, digest)
	if err != nil {
		return errors.Wrapf(err, "can not delete repository manifest %s:%s (%s)", deleteTag.Repository, deleteTag.Tag, digest)
	}

	return nil
}

// Final message.
func (p *Provider) PostCommand() error {
	log.Infof("Done")

	return nil
}
