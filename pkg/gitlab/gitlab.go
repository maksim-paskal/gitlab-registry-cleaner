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
package gitlab

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/utils"
	"github.com/pkg/errors"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var (
	gitlabToken = flag.String("gitlab.token", os.Getenv("GITLAB_TOKEN"), "")
	gitlabURL   = flag.String("gitlab.url", os.Getenv("GITLAB_URL"), "")
)

const (
	// max list size for gitlab api.
	gilabAPIMaxListSize = 100
	// hours in day.
	hoursInDay = 24
	// delete docker tag if last commit more than 30 days ago.
	staleBranchDays = 30
)

var git *gitlab.Client

// Gitlab create new client.
func Init() error {
	var err error

	git, err = gitlab.NewClient(*gitlabToken, gitlab.WithBaseURL(*gitlabURL))
	if err != nil {
		return errors.Wrap(err, "can not connect to gitlab")
	}

	return nil
}

// Return project ID on docker repo.
func GetProjectID(ctx context.Context, dockerRepo string) (int, error) {
	gitlabProject, _, err := git.Projects.GetProject(
		dockerRepo,
		&gitlab.GetProjectOptions{},
		gitlab.WithContext(ctx))
	if err != nil {
		return 0, errors.Wrap(err, dockerRepo)
	}

	return gitlabProject.ID, nil
}

type GetProjectBranchesResult struct {
	Staled             bool
	StaledDays         int
	OriginalBranchName string
}

// Return all gitlab branches slugnames with bool stage flag.
func GetProjectBranches(ctx context.Context, projectID int) (map[string]*GetProjectBranchesResult, error) {
	result := make(map[string]*GetProjectBranchesResult)

	currentPage := 0

	for {
		if ctx.Err() != nil {
			return nil, errors.Wrap(ctx.Err(), "context error")
		}

		currentPage++

		gitBranches, _, err := git.Branches.ListBranches(
			projectID,
			&gitlab.ListBranchesOptions{
				ListOptions: gitlab.ListOptions{
					Page:    currentPage,
					PerPage: gilabAPIMaxListSize,
				},
			},
			gitlab.WithContext(ctx),
		)
		if err != nil {
			return nil, errors.Wrap(err, "can not list branches")
		}

		if len(gitBranches) == 0 {
			break
		}

		for _, gitBranch := range gitBranches {
			lastCommitHoursAgo := time.Since(*gitBranch.Commit.CommittedDate).Hours()
			branchSlug := utils.GitlabSluglify(gitBranch.Name)

			item := GetProjectBranchesResult{
				StaledDays:         staleBranchDays,
				OriginalBranchName: gitBranch.Name,
			}

			if lastCommitHoursAgo > hoursInDay*staleBranchDays {
				item.Staled = true
			}

			result[branchSlug] = &item
		}
	}

	return result, nil
}
