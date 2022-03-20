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
package providers

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	registryURL       = flag.String("registry.url", "http://127.0.0.1:5000", "")
	registryLocalPath = flag.String("registry.path", "/var/lib/registry/", "")
	deleteCommand     strings.Builder
)

const (
	resultFile           = "cleanOldTags.sh"
	resultFilePermission = 0o744
)

type LocalRegistry struct{}

var hub *registry.Registry

// Create new client.
func (*LocalRegistry) Init() error {
	var err error

	hub, err = registry.New(*registryURL, "", "")
	if err != nil {
		return errors.Wrap(err, "can not connect to registry")
	}

	hub.Logf = registry.Quiet

	deleteCommand.WriteString("#!/bin/sh\n")

	return nil
}

// List repositories.
func (*LocalRegistry) Repositories() ([]string, error) {
	repos, err := hub.Repositories()

	return repos, errors.Wrap(err, "can not get repositories")
}

// List tags.
func (*LocalRegistry) Tags(repository string) ([]string, error) {
	tags, err := hub.Tags(repository)

	return tags, errors.Wrap(err, "can not get tags")
}

// Delete tag.
func (*LocalRegistry) DeleteTag(repository string, tag string, tagType types.TagType) error {
	log.Debugf("%s:%s (%d)", repository, tag, tagType)

	command := fmt.Sprintf("rm -rf %sdocker/registry/v2/repositories/%s/_manifests/tags/%s # reason=%d\n",
		*registryLocalPath,
		repository,
		tag,
		tagType)

	deleteCommand.WriteString(command)

	return nil
}

// Create file.
func (*LocalRegistry) PostCommand() error {
	err := ioutil.WriteFile(resultFile, []byte(deleteCommand.String()), resultFilePermission)
	if err != nil {
		return errors.Wrap(err, "can not write file")
	}

	log.Infof("%s created", resultFile)

	return nil
}
