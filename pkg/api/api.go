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
package api

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	hoursInDay                  = 24
	slashesInDockerRegistryPath = 3
)

var version = "dev"

// Get application version.
func GetVersion() string {
	return version
}

// Convert docker registry path to gitlab path.
func GetGitlabProjectPath(dockerRegistryPath string) (string, error) {
	pathGroups := strings.Split(dockerRegistryPath, "/")

	if len(pathGroups) < slashesInDockerRegistryPath {
		return "", errors.Errorf("path %s must contain group/project/image path", dockerRegistryPath)
	}

	if len(pathGroups) > 0 {
		pathGroups = pathGroups[:len(pathGroups)-1]
	}

	return strings.Join(pathGroups, "/"), nil
}

type GetNotDeletableTagsInput struct {
	Tags             map[string]types.TagType
	DateRegexp       *regexp.Regexp
	NotDeleteDays    float64
	MinNotDeleteTags int
}

// Detect stale tag.
func GetNotDeletableTags(input *GetNotDeletableTagsInput) []string { //nolint:cyclop
	tagsNotToDelete := make([]string, 0)
	allTagDate := make([]string, 0)
	tagDateMaxDate := time.Time{}

	// Detect max release date
	for tag := range input.Tags {
		if input.DateRegexp.MatchString(tag) {
			tagDate, err := time.Parse("20060102", input.DateRegexp.FindStringSubmatch(tag)[1])
			if err != nil {
				log.WithError(err).Error()

				continue
			}

			if time.Since(tagDate) < 0 {
				log.Errorf("release date %s in future - ignore", tag)

				continue
			}

			if tagDate.After(tagDateMaxDate) {
				tagDateMaxDate = tagDate
			}

			allTagDate = append(allTagDate, tag)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(allTagDate)))

	// Detect days between tag and maxrelease date
	// if diff > 10 days - tag will be removed
	for _, tag := range allTagDate {
		tagDate, _ := time.Parse("20060102", input.DateRegexp.FindStringSubmatch(tag)[1])

		dateDiffDays := tagDateMaxDate.Sub(tagDate).Hours() / hoursInDay

		log.Debugf("%s, datediff=%f", tag, dateDiffDays)

		if dateDiffDays < input.NotDeleteDays {
			tagsNotToDelete = append(tagsNotToDelete, tag)
		}
	}

	// leave last 3 tags if final tagsNotToDelete is less of this amount
	if len(tagsNotToDelete) < input.MinNotDeleteTags {
		latestTags := make([]string, 0)
		for i := 0; i < len(allTagDate); i++ {
			latestTags = append(latestTags, allTagDate[i])

			if (len(latestTags)) >= input.MinNotDeleteTags {
				break
			}
		}

		tagsNotToDelete = latestTags
	}

	return tagsNotToDelete
}
