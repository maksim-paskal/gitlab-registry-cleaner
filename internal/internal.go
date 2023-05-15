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
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/api"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/gitlab"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/metrics"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/providers/docker"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/providers/s3"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/types"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/utils"
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
	snapshotRepositoryPattern = flag.String("snapshot.repository", utils.GetEnv("SNAPSHOT_REPOSITORY", `^devops/docker/mysql-.+$`), "") //nolint:lll
	snapshotTagPattern        = flag.String("snapshot.tag", utils.GetEnv("SNAPSHOT_TAG", `^(\d{8})-snap$`), "")
	snapshotNotDeleteDays     = flag.Float64("snapshot.daysNotDelete", defaultNotDeleteDays, "")
	minNotDeleteSnapshotTags  = flag.Int("snapshot.minTags", defaultMinNotDeleteTags, "")
	registryFilter            = flag.String("registry.filter", "", "")
	releaseTagPattern         = flag.String("release.tag", utils.GetEnv("RELEASE_TAG", `^release-(\d{8}).*$`), "")
	systemTagPattern          = flag.String("system.tag", utils.GetEnv("SYSTEM_TAG", `^(main|master)$`), "")
	ignoreRepositoryPattern   = flag.String("ignoreTags", utils.GetEnv("IGNORE_TAGS", `^devops/docker$`), "")
	releaseNotDeleteDays      = flag.Float64("release.daysNotDelete", defaultNotDeleteDays, "")
	minNotDeleteReleaseTags   = flag.Int("release.minTags", defaultMinNotDeleteTags, "")
	ciCheck                   = flag.Bool("ci.check", false, "check if release tag is valid")
	ciTag                     = flag.String("ci.tag", os.Getenv("CI_COMMIT_REF_NAME"), "tag to check")
	ciCommitDate              = flag.String("ci.commitDate", os.Getenv("CI_COMMIT_TIMESTAMP"), "commit date to check")
)

var releaseTagRegexp,
	systemTagRegexp,
	ignoreRepositoryRegexp,
	snapshotRepositoryRegexp,
	snapshotTagRegexp *regexp.Regexp

func Init() {
	releaseTagRegexp = regexp.MustCompile(*releaseTagPattern)
	systemTagRegexp = regexp.MustCompile(*systemTagPattern)
	ignoreRepositoryRegexp = regexp.MustCompile(*ignoreRepositoryPattern)
	snapshotRepositoryRegexp = regexp.MustCompile(*snapshotRepositoryPattern)
	snapshotTagRegexp = regexp.MustCompile(*snapshotTagPattern)
}

// Run main logic.
func Run(ctx context.Context) error { //nolint:funlen,cyclop
	// Login to gitlab
	if err := gitlab.Init(); err != nil {
		log.Fatal(err)
	}

	if *ciCheck {
		if err := api.CheckReleaseTag(releaseTagRegexp, *ciTag, *ciCommitDate); err != nil {
			fmt.Printf("Tag %s is not valid:\n%s", *ciTag, err.Error()) //nolint:forbidigo

			os.Exit(1)
		}

		fmt.Printf("Tag %s is valid\n", *ciTag) //nolint:forbidigo

		return nil
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
	if err := registry.Init(ctx, *dryRun); err != nil {
		return errors.Wrap(err, "can not init registry")
	}

	// get all docker repository
	repositories, err := registry.Repositories(ctx, *registryFilter)
	if err != nil {
		return errors.Wrap(err, "can not list repositories")
	}

	log.Infof("repositories: %v", repositories)

	tagsToDelete := make([]types.DeleteTagInput, 0)

	// get stalled docker tags
	staledDockerTags, err := getStaleDockerTags(ctx, registry, repositories)
	if err != nil {
		return errors.Wrap(err, "can not get staled docker tags")
	}

	tagsToDelete = append(tagsToDelete, staledDockerTags...)

	// get staled snapshot tags
	if *snapshotEnabled {
		tagsToDelete = append(tagsToDelete, getStaledSnashotsTags(ctx, registry, repositories)...)
	}

	// delete tags from registry
	for _, tag := range tagsToDelete {
		metrics.TagsDeleted.Inc()
		log.Infof("delete image=%s:%s reason=%s", tag.Repository, tag.Tag, tag.TagType.String())

		// tag will be removed
		err := registry.DeleteTag(ctx, tag)
		if err != nil {
			metrics.TagsErrors.Inc()
			log.WithError(err).Errorf("%s:%s reason=%s", tag.Repository, tag.Tag, tag.TagType.String())
		}
	}

	// Run post commands in registry
	if err := registry.PostCommand(ctx); err != nil {
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
	if err := metrics.Push(ctx); err != nil {
		return errors.Wrap(err, "can not process metrics push")
	}

	return nil
}

// get staled docker tags to delete from docker registry.
func getStaleDockerTags(ctx context.Context, registry types.Provider, repositories []string) ([]types.DeleteTagInput, error) { //nolint:funlen,gocognit,lll,cyclop
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
		gitlabProjectID, err := gitlab.GetProjectID(ctx, gitlabRepo)
		if err != nil {
			log.WithError(err).Error(gitlabRepo)

			continue
		}

		log.Debugf("gitlab repositories %s %d %v", gitlabRepo, gitlabProjectID, dockerRepos)

		projectBranches, err := gitlab.GetProjectBranches(ctx, gitlabProjectID)
		if err != nil {
			return tagsToDelete, errors.Wrap(err, "can not get branches")
		}

		log.Debugf("projectBranches %v", projectBranches)

		projectAllDockerTags := make(map[string]types.TagType)

		// Get docker tags
		for _, dockerRepo := range dockerRepos {
			dockerTags, _ := registry.Tags(ctx, dockerRepo)
			for _, dockerTag := range dockerTags {
				projectAllDockerTags[dockerTag] = types.Unknown
			}
		}

		tagsNotToDelete := api.GetNotDeletableTags(&api.GetNotDeletableTagsInput{
			Tags:             projectAllDockerTags,
			DateRegexp:       releaseTagRegexp,
			NotDeleteDays:    *releaseNotDeleteDays,
			MinNotDeleteTags: *minNotDeleteReleaseTags,
		})

		// Calculate tags to delete
		for projectAllDockerTag := range projectAllDockerTags {
			var tagType types.TagType

			// remove arch from docker tag name
			tagWithoutArch := api.GetTagWithoutArch(projectAllDockerTag)

			if branchStale, ok := projectBranches[tagWithoutArch]; ok {
				if branchStale.Staled {
					tagType = types.BranchStale
				} else {
					tagType = types.BranchNotStaled

					log.Debugf("%s branch %s (%s) has last commit less than %d days",
						gitlabRepo,
						branchStale.OriginalBranchName,
						tagWithoutArch,
						branchStale.StaledDays,
					)
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

			if systemTagRegexp.MatchString(tagWithoutArch) {
				tagType = types.SystemTag
			}

			if len(tagType) == 0 {
				tagType = types.Unknown
			}

			projectAllDockerTags[projectAllDockerTag] = tagType
		}

		// List all tags
		for _, dockerRepo := range dockerRepos {
			dockerTags, _ := registry.Tags(ctx, dockerRepo)
			for _, dockerTag := range dockerTags {
				tagType := projectAllDockerTags[dockerTag]

				switch tagType { //nolint:exhaustive
				case types.ReleaseTag, types.BranchNotFound, types.BranchStale:
					tagsToDelete = append(tagsToDelete, types.DeleteTagInput{
						Repository: dockerRepo,
						Tag:        dockerTag,
						TagType:    tagType,
					})
				case types.Unknown:
					log.Warnf("%s:%s,%s", dockerRepo, dockerTag, tagType)
				default:
					log.Infof("%s:%s,%s", dockerRepo, dockerTag, tagType)
				}
			}
		}
	}

	return tagsToDelete, nil
}

// get staled snapshots tags to delete from docker registry.
func getStaledSnashotsTags(ctx context.Context, registry types.Provider, repositories []string) []types.DeleteTagInput {
	tagsToDelete := make([]types.DeleteTagInput, 0)

	for _, dockerRepo := range repositories {
		if snapshotRepositoryRegexp.MatchString(dockerRepo) {
			snapshotsDockerTags := make(map[string]types.TagType)
			dockerTags, _ := registry.Tags(ctx, dockerRepo)

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
