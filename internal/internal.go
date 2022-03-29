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
package internal

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/gitlab"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/metrics"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/providers/docker"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/utils"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
)

const (
	hoursInDay                     = 24
	defaultReleaseNotDeleteDays    = 10
	defaultMinNotDeleteReleaseTags = 3
	slashesInDockerRegistryPath    = 3
)

var (
	provider                = flag.String("provider", "docker", "")
	dryRun                  = flag.Bool("dry-run", false, "")
	releaseTagPattern       = flag.String("release.tag", `^release-(\d{4}\d{2}\d{2}).*$`, "")
	systemTagPattern        = flag.String("system.tag", `^main$|^master$`, "")
	ignoreRepositoryPattern = flag.String("ignoreTags", `^devops/docker$`, "")
	releaseNotDeleteDays    = flag.Float64("release.daysNotDelete", defaultReleaseNotDeleteDays, "")
	minNotDeleteReleaseTags = flag.Int("release.minTags", defaultMinNotDeleteReleaseTags, "")
)

var (
	releaseTagRegexp       = regexp.MustCompile(*releaseTagPattern)
	systemTagRegexp        = regexp.MustCompile(*systemTagPattern)
	ignoreRepositoryRegexp = regexp.MustCompile(*ignoreRepositoryPattern)
)

func Run() error { //nolint:funlen,gocognit,cyclop,gocyclo
	// Login to gitlab
	if err := gitlab.Init(); err != nil {
		log.Fatal(err)
	}

	var registry types.Provider

	switch *provider {
	case "docker":
		registry = &docker.Provider{}
	default:
		return errors.Errorf("%s unknown provider", *provider)
	}

	log.Infof("Starting %s %s...", filepath.Base(os.Args[0]), GetVersion())
	log.Infof("Using %s provider...", *provider)

	// Login to registry
	if err := registry.Init(); err != nil {
		return errors.Wrap(err, "can not init registry")
	}

	repositories, err := registry.Repositories()
	if err != nil {
		return errors.Wrap(err, "can not list repositories")
	}

	gitlabProjects := make(map[string][]string)

	// Convert docker path to gitlab project path
	for _, repo := range repositories {
		log.Debug("docker repositories", repo)

		gitlabProjectPath, err := GetGitlabProjectPath(repo)
		if err != nil {
			log.Warn(err)
			metrics.TagsWarnings.Inc()

			continue
		}

		// ignore some projects
		if ignoreRepositoryRegexp.MatchString(gitlabProjectPath) {
			continue
		}

		// create unique gitlab projects
		if gitlabProjects[gitlabProjectPath] == nil {
			gitlabProjects[gitlabProjectPath] = []string{repo}
		} else {
			gitlabProjects[gitlabProjectPath] = append(gitlabProjects[gitlabProjectPath], repo)
		}
	}

	// For all gitlab project list branch and detect stale docker tag
	for gitlabRepo, dockerRepos := range gitlabProjects {
		gitlabProjectID, err := gitlab.GetProjectID(gitlabRepo)
		if err != nil {
			log.Error(errors.Wrap(err, gitlabRepo))

			continue
		}

		log.Debugf("gitlab repositories %s %d %v", gitlabRepo, gitlabProjectID, dockerRepos)

		projectBranches, err := gitlab.GetProjectBranches(gitlabProjectID)
		if err != nil {
			return errors.Wrap(err, "can not get branches")
		}

		projectAllDockerTags := make(map[string]types.TagType)

		// Get docker tags
		for _, dockerRepo := range dockerRepos {
			dockerTags, _ := registry.Tags(dockerRepo)
			for _, dockerTag := range dockerTags {
				projectAllDockerTags[dockerTag] = types.CanNotDelete
			}
		}

		tagsNotToDelete := GetNotDeletableReleaseTags(projectAllDockerTags)

		// Calculate tags to delete
		for projectAllDockerTag, tagType := range projectAllDockerTags {
			if branchStale, ok := projectBranches[projectAllDockerTag]; ok {
				if branchStale {
					tagType = types.BranchStale
				}
			} else {
				tagType = types.BranchNotFound
			}

			if releaseTagRegexp.MatchString(projectAllDockerTag) {
				if utils.StringInSlice(projectAllDockerTag, tagsNotToDelete) {
					tagType = types.ReleaseTagCanNotDelete
				} else {
					tagType = types.ReleaseTag
				}
			}

			if systemTagRegexp.MatchString(projectAllDockerTag) {
				tagType = types.SystemTag
			}

			projectAllDockerTags[projectAllDockerTag] = tagType
		}

		// List all tags
		for _, dockerRepo := range dockerRepos {
			dockerTags, _ := registry.Tags(dockerRepo)
			for _, dockerTag := range dockerTags {
				tagType := projectAllDockerTags[dockerTag]

				if tagType == types.ReleaseTag || tagType == types.BranchNotFound || tagType == types.BranchStale {
					if *dryRun {
						log.Infof("Not deleting tag, dry-run mode, image=%s:%s reason=%s", dockerRepo, dockerTag, tagType.String())

						continue
					}

					// tags will be removed
					err := registry.DeleteTag(dockerRepo, dockerTag, tagType)
					if err != nil {
						return errors.Wrap(err, "can not delete tag")
					}

					metrics.TagsDeleted.Inc()
				}
			}
		}
	}

	// Run post commands in registry
	if err := registry.PostCommand(); err != nil {
		return errors.Wrap(err, "can not process post command")
	}

	tagsDeleted := &dto.Metric{}
	if err := metrics.TagsDeleted.Write(tagsDeleted); err != nil {
		return errors.Wrap(err, "can not get current deleted tags")
	}

	tagsWarnings := &dto.Metric{}
	if err := metrics.TagsWarnings.Write(tagsWarnings); err != nil {
		return errors.Wrap(err, "can not get current warnings tags")
	}

	log.Infof("tags deleted %s warnings %s", tagsDeleted.Counter.String(), tagsWarnings.Counter.String())

	metrics.CompletionTime.SetToCurrentTime()

	// send metrics to pushgateway
	if err := metrics.Push(); err != nil {
		return errors.Wrap(err, "can not process metrics push")
	}

	return nil
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

// Detect stale release tag.
func GetNotDeletableReleaseTags(projectAllDockerTags map[string]types.TagType) []string { //nolint:funlen,cyclop
	tagsNotToDelete := make([]string, 0)
	allReleaseTags := make([]string, 0)
	releaseMaxDate := time.Time{}

	// Detect max release date
	for projectAllDockerTag := range projectAllDockerTags {
		if releaseTagRegexp.MatchString(projectAllDockerTag) {
			releaseDate, err := time.Parse("20060102", releaseTagRegexp.FindStringSubmatch(projectAllDockerTag)[1])
			if err != nil {
				log.Error(err)

				continue
			}

			if time.Since(releaseDate) < 0 {
				log.Errorf("release date %s in future - ignore", projectAllDockerTag)

				continue
			}

			if releaseDate.After(releaseMaxDate) {
				releaseMaxDate = releaseDate
			}

			allReleaseTags = append(allReleaseTags, projectAllDockerTag)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(allReleaseTags)))

	// Detect days between tag and maxrelease date
	// if diff > 10 days - tag will be removed
	for _, tag := range allReleaseTags {
		releaseDate, err := time.Parse("20060102", releaseTagRegexp.FindStringSubmatch(tag)[1])
		if err != nil {
			log.Error(err)

			continue
		}

		releaseDateDiffDays := releaseMaxDate.Sub(releaseDate).Hours() / hoursInDay

		log.Debugf("%s, datediff=%f", tag, releaseDateDiffDays)

		if releaseDateDiffDays < *releaseNotDeleteDays {
			tagsNotToDelete = append(tagsNotToDelete, tag)
		}
	}

	// leave last 3 tags if final tagsNotToDelete is less of this amount
	if len(tagsNotToDelete) < *minNotDeleteReleaseTags {
		latestTags := make([]string, 0)
		for i := 0; i < len(allReleaseTags); i++ {
			latestTags = append(latestTags, allReleaseTags[i])

			if (len(latestTags)) >= *minNotDeleteReleaseTags {
				break
			}
		}

		tagsNotToDelete = latestTags
	}

	return tagsNotToDelete
}

var version = "dev"

func GetVersion() string {
	return version
}
