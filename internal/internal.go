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

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/api"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/gitlab"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/metrics"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/providers/docker"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/providers/s3"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/utils"
	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	log "github.com/sirupsen/logrus"
)

const (
	defaultNotDeleteDays    = 10
	defaultMinNotDeleteTags = 3
)

var (
	provider                  = flag.String("provider", "docker", "")
	dryRun                    = flag.Bool("dry-run", false, "")
	snapshotEnabled           = flag.Bool("snapshots", false, "enable snapshot clearing")
	snapshotRepositoryPattern = flag.String("snapshot.repository", `^devops/docker/mysql-.+$`, "")
	snapshotTagPattern        = flag.String("snapshot.tag", `^(\d{8})-snap$`, "")
	snapshotNotDeleteDays     = flag.Float64("snapshot.daysNotDelete", defaultNotDeleteDays, "")
	minNotDeleteSnapshotTags  = flag.Int("snapshot.minTags", defaultMinNotDeleteTags, "")
	releaseTagPattern         = flag.String("release.tag", `^release-(\d{8}).*$`, "")
	systemTagPattern          = flag.String("system.tag", `^main$|^master$`, "")
	ignoreRepositoryPattern   = flag.String("ignoreTags", `^devops/docker$`, "")
	releaseNotDeleteDays      = flag.Float64("release.daysNotDelete", defaultNotDeleteDays, "")
	minNotDeleteReleaseTags   = flag.Int("release.minTags", defaultMinNotDeleteTags, "")
)

var (
	releaseTagRegexp         = regexp.MustCompile(*releaseTagPattern)
	systemTagRegexp          = regexp.MustCompile(*systemTagPattern)
	ignoreRepositoryRegexp   = regexp.MustCompile(*ignoreRepositoryPattern)
	snapshotRepositoryRegexp = regexp.MustCompile(*snapshotRepositoryPattern)
	snapshotTagRegexp        = regexp.MustCompile(*snapshotTagPattern)
)

// Run main logic.
func Run() error { //nolint:funlen,cyclop
	// Login to gitlab
	if err := gitlab.Init(); err != nil {
		log.Fatal(err)
	}

	var registry types.Provider

	switch *provider {
	case "docker":
		registry = &docker.Provider{}
	case "s3":
		registry = &s3.Provider{}
	default:
		return errors.Errorf("%s unknown provider", *provider)
	}

	log.Infof("Starting %s %s...", filepath.Base(os.Args[0]), api.GetVersion())
	log.Infof("Using %s provider...", *provider)

	// Login to registry
	if err := registry.Init(*dryRun); err != nil {
		return errors.Wrap(err, "can not init registry")
	}

	// get all docker repository
	repositories, err := registry.Repositories()
	if err != nil {
		return errors.Wrap(err, "can not list repositories")
	}

	tagsToDelete := make([]types.DeleteTagInput, 0)

	// get stalled docker tags
	staledDockerTags, err := getStaleDockerTags(registry, repositories)
	if err != nil {
		return errors.Wrap(err, "can not get staled docker tags")
	}

	tagsToDelete = append(tagsToDelete, staledDockerTags...)

	// get staled snapshot tags
	if *snapshotEnabled {
		tagsToDelete = append(tagsToDelete, getStaledSnashotsTags(registry, repositories)...)
	}

	// delete tags from registry
	for _, tag := range tagsToDelete {
		metrics.TagsDeleted.Inc()
		log.Infof("delete image=%s:%s reason=%s", tag.Repository, tag.Tag, tag.TagType.String())

		// tag will be removed
		err := registry.DeleteTag(tag)
		if err != nil {
			metrics.TagsErrors.Inc()
			log.WithError(err).Errorf("%s:%s reason=%s", tag.Repository, tag.Tag, tag.TagType.String())
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

	tagsErrors := &dto.Metric{}
	if err := metrics.TagsErrors.Write(tagsErrors); err != nil {
		return errors.Wrap(err, "can not get current errors tags")
	}

	log.Infof("tags deleted %s warnings %s errors %s",
		tagsDeleted.Counter.String(),
		tagsWarnings.Counter.String(),
		tagsErrors.Counter.String(),
	)

	metrics.CompletionTime.SetToCurrentTime()

	// send metrics to pushgateway
	if err := metrics.Push(); err != nil {
		return errors.Wrap(err, "can not process metrics push")
	}

	return nil
}

// get staled docker tags to delete from docker registry.
func getStaleDockerTags(registry types.Provider, repositories []string) ([]types.DeleteTagInput, error) { //nolint:funlen,gocognit,lll,cyclop
	tagsToDelete := make([]types.DeleteTagInput, 0)
	gitlabProjects := make(map[string][]string)

	// Convert docker path to gitlab project path
	for _, repo := range repositories {
		log.Debug("docker repositories", repo)

		gitlabProjectPath, err := api.GetGitlabProjectPath(repo)
		if err != nil {
			log.WithError(err).Warn()
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
			log.WithError(err).Error(gitlabRepo)

			continue
		}

		log.Debugf("gitlab repositories %s %d %v", gitlabRepo, gitlabProjectID, dockerRepos)

		projectBranches, err := gitlab.GetProjectBranches(gitlabProjectID)
		if err != nil {
			return tagsToDelete, errors.Wrap(err, "can not get branches")
		}

		projectAllDockerTags := make(map[string]types.TagType)

		// Get docker tags
		for _, dockerRepo := range dockerRepos {
			dockerTags, _ := registry.Tags(dockerRepo)
			for _, dockerTag := range dockerTags {
				projectAllDockerTags[dockerTag] = types.CanNotDelete
			}
		}

		tagsNotToDelete := api.GetNotDeletableTags(&api.GetNotDeletableTagsInput{
			Tags:             projectAllDockerTags,
			DateRegexp:       releaseTagRegexp,
			NotDeleteDays:    *releaseNotDeleteDays,
			MinNotDeleteTags: *minNotDeleteReleaseTags,
		})

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
					tagsToDelete = append(tagsToDelete, types.DeleteTagInput{
						Repository: dockerRepo,
						Tag:        dockerTag,
						TagType:    tagType,
					})
				}
			}
		}
	}

	return tagsToDelete, nil
}

// get staled snapshots tags to delete from docker registry.
func getStaledSnashotsTags(registry types.Provider, repositories []string) []types.DeleteTagInput {
	tagsToDelete := make([]types.DeleteTagInput, 0)

	for _, dockerRepo := range repositories {
		if snapshotRepositoryRegexp.MatchString(dockerRepo) {
			snapshotsDockerTags := make(map[string]types.TagType)
			dockerTags, _ := registry.Tags(dockerRepo)

			// get all repository tags
			for _, dockerTag := range dockerTags {
				snapshotsDockerTags[dockerTag] = types.SnapshotStaled
			}

			tagsNotToDelete := api.GetNotDeletableTags(&api.GetNotDeletableTagsInput{
				Tags:             snapshotsDockerTags,
				DateRegexp:       snapshotTagRegexp,
				NotDeleteDays:    *snapshotNotDeleteDays,
				MinNotDeleteTags: *minNotDeleteSnapshotTags,
			})

			// Calculate tags to delete
			for snapshotsDockerTag, tagType := range snapshotsDockerTags {
				if utils.StringInSlice(snapshotsDockerTag, tagsNotToDelete) {
					tagType = types.SnapshotTagCanNotDelete
				}

				if tagType == types.SnapshotStaled {
					tagsToDelete = append(tagsToDelete, types.DeleteTagInput{
						Repository: dockerRepo,
						Tag:        snapshotsDockerTag,
						TagType:    tagType,
					})
				}
			}
		}
	}

	return tagsToDelete
}
