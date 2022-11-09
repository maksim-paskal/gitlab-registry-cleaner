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
	"flag"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var checkReleseTagDelta = flag.Int("ci.releases-delta-days", defaultCheckReleseTagDelta, "number of days allowed to be between release tag and commit date") //nolint:lll

const (
	hoursInDay                  = 24
	slashesInDockerRegistryPath = 3
	defaultCheckReleseTagDelta  = 5
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
func GetNotDeletableTags(input *GetNotDeletableTagsInput) []string {
	tagsNotToDelete := make([]string, 0)
	allTagDate := make([]string, 0)
	tagDateMaxDate := time.Time{}

	// Detect max release date
	for tag := range input.Tags {
		releaseTag, err := GetReleaseTag(input.DateRegexp, tag)
		if err != nil {
			log.WithError(err).Error()

			continue
		}

		if releaseTag.TagDate.After(tagDateMaxDate) {
			tagDateMaxDate = releaseTag.TagDate
		}

		allTagDate = append(allTagDate, tag)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(allTagDate)))

	// Detect days between tag and maxrelease date
	// if diff > 10 days - tag will be removed
	for _, tag := range allTagDate {
		releaseTag, err := GetReleaseTag(input.DateRegexp, tag)
		if err != nil {
			log.WithError(err).Error()

			continue
		}

		dateDiffDays := tagDateMaxDate.Sub(releaseTag.TagDate).Hours() / hoursInDay

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

func GetTagWithoutArch(tagName string, tagArch []string) string {
	formatedTag := tagName

	for _, arch := range tagArch {
		suffix := fmt.Sprintf("-%s", arch)
		formatedTag = strings.TrimSuffix(formatedTag, suffix)
	}

	return formatedTag
}

type ReleaseTag struct {
	TagName string
	TagDate time.Time
}

func GetReleaseTag(tagNameRegexp *regexp.Regexp, tagName string) (*ReleaseTag, error) {
	if !tagNameRegexp.MatchString(tagName) {
		return nil, fmt.Errorf("tag %s doesn't match regexp", tagName) //nolint:goerr113
	}

	tagDate, err := time.Parse("20060102", tagNameRegexp.FindStringSubmatch(tagName)[1])
	if err != nil {
		return nil, errors.Wrap(err, "can not parse date")
	}

	if time.Since(tagDate) < 0 {
		return nil, fmt.Errorf("tag date can not be in future") //nolint:goerr113
	}

	return &ReleaseTag{
		TagName: tagName,
		TagDate: tagDate,
	}, nil
}

// check if release tag has valid format, date in tag must be +- 5 days from commit date.
func CheckReleaseTag(tagNameRegexp *regexp.Regexp, tagName string, commitDate string) error {
	releaseTag, err := GetReleaseTag(tagNameRegexp, tagName)
	if err != nil {
		return errors.Wrap(err, "can not get release tag")
	}

	commitDateTime, err := time.Parse(time.RFC3339, commitDate)
	if err != nil {
		return errors.Wrap(err, "parse commit date")
	}

	commitDateDiff := commitDateTime.Sub(releaseTag.TagDate)

	commitDateDiffDays := math.Abs(commitDateDiff.Hours() / hoursInDay)

	if commitDateDiffDays > float64(*checkReleseTagDelta) {
		return fmt.Errorf("difference between commit date and release bigger than %d days", *checkReleseTagDelta) //nolint:goerr113,lll
	}

	return nil
}
