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

	utilsgo "github.com/maksim-paskal/utils-go"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/gitlab"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/providers"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var provider = flag.String("provider", "local", "")

const (
	releaseTagPattern       = `^release-(\d{4}\d{2}\d{2}).*$`
	systemTagPattern        = `^main$|^master$`
	ignoreRepositoryPattern = `^devops/docker$`
	hoursInDay              = 24
	releaseNotDeleteDays    = 10
	minNotDeleteReleaseTags = 3
)

var (
	releaseTagRegexp       = regexp.MustCompile(releaseTagPattern)
	systemTagRegexp        = regexp.MustCompile(systemTagPattern)
	ignoreRepositoryRegexp = regexp.MustCompile(ignoreRepositoryPattern)
)

func CleanOldTags() error { //nolint:funlen,gocognit,cyclop
	// Login to gitlab
	if err := gitlab.Init(); err != nil {
		log.Fatal(err)
	}

	var registry types.Provider

	switch *provider {
	case "local":
		registry = &providers.LocalRegistry{}
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
		gitlabProjectPath := GetGitlabProjectPath(repo)

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
				if utilsgo.StringInSlice(projectAllDockerTag, tagsNotToDelete) {
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
					// tags will be removed
					err := registry.DeleteTag(dockerRepo, dockerTag, tagType)
					if err != nil {
						return errors.Wrap(err, "can not delete tag")
					}
				}
			}
		}
	}

	// Run post commands in registry
	if err := registry.PostCommand(); err != nil {
		return errors.Wrap(err, "can not process post command")
	}

	return nil
}

// Convert docker registry path to gitlab path.
func GetGitlabProjectPath(dockerRegistryPath string) string {
	pathGroups := strings.Split(dockerRegistryPath, "/")
	if len(pathGroups) > 0 {
		pathGroups = pathGroups[:len(pathGroups)-1]
	}

	return strings.Join(pathGroups, "/")
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

		if releaseDateDiffDays < releaseNotDeleteDays {
			tagsNotToDelete = append(tagsNotToDelete, tag)
		}
	}

	// leave last 3 tags if final tagsNotToDelete is less of this amount
	if len(tagsNotToDelete) < minNotDeleteReleaseTags {
		latestTags := make([]string, 0)
		for i := 0; i < len(allReleaseTags); i++ {
			latestTags = append(latestTags, allReleaseTags[i])

			if (len(latestTags)) >= minNotDeleteReleaseTags {
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
