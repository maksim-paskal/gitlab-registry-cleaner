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
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	registryURL      = flag.String("registry.url", os.Getenv("REGISTRY_URL"), "format https://registry.com")
	registryLogin    = flag.String("registry.username", os.Getenv("REGISTRY_USERNAME"), "")
	registryPassword = flag.String("registry.password", os.Getenv("REGISTRY_PASSWORD"), "")
)

type Provider struct {
	hub *registry.Registry
}

// Create new client.
func (p *Provider) Init() error {
	var err error

	p.hub, err = registry.New(*registryURL, *registryLogin, *registryPassword)
	if err != nil {
		return errors.Wrap(err, "can not connect to registry")
	}

	p.hub.Logf = registry.Quiet

	return nil
}

// List repositories.
func (p *Provider) Repositories() ([]string, error) {
	repos, err := p.hub.Repositories()

	return repos, errors.Wrap(err, "can not get repositories")
}

// List tags.
func (p *Provider) Tags(repository string) ([]string, error) {
	tags, err := p.hub.Tags(repository)

	return tags, errors.Wrap(err, "can not get tags")
}

// Delete tag.
func (p *Provider) DeleteTag(repository string, tag string, tagType types.TagType) error {
	log.Infof("deleting %s:%s", repository, tag)

	digest, err := p.manifestDigest(repository, tag)
	if err != nil {
		return errors.Wrap(err, "can not get digest")
	}

	err = p.hub.DeleteManifest(repository, digest)
	if err != nil {
		return errors.Wrapf(err, "can not delete repository manifest %s:%s (%s)", repository, tag, digest)
	}

	return nil
}

// Final message.
func (p *Provider) PostCommand() error {
	log.Infof("Done")

	return nil
}

// fix registry client header
// https://github.com/heroku/docker-registry-client/pull/79
func (p *Provider) manifestDigest(repository, reference string) (digest.Digest, error) {
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", *registryURL, repository, reference)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", errors.Wrap(err, "can not create request")
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := p.hub.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", errors.Wrap(err, "can not create Do")
	}

	d, err := digest.Parse(resp.Header.Get("Docker-Content-Digest"))
	if err != nil {
		return "", errors.Wrap(err, "can not parse header")
	}

	return d, nil
}
