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
	"flag"
	"os"
	"time"

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/utils"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
)

var (
	gitlabToken = flag.String("gitlab.token", os.Getenv("GITLAB_TOKEN"), "")
	gitlabURL   = flag.String("gitlab.url", os.Getenv("GITLAB_URL"), "")
)

const (
	gilabAPIMaxListSize = 100
	hoursInDay          = 24
	staleBranchDays     = 100
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
func GetProjectID(dockerRepo string) (int, error) {
	gitlabProject, _, err := git.Projects.GetProject(dockerRepo, &gitlab.GetProjectOptions{})
	if err != nil {
		return 0, errors.Wrap(err, dockerRepo)
	}

	return gitlabProject.ID, nil
}

// Return all gitlab branches slugnames with bool stage flag.
func GetProjectBranches(projectID int) (map[string]bool, error) {
	result := make(map[string]bool)

	currentPage := 0

	for {
		gitBranches, _, err := git.Branches.ListBranches(projectID, &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    currentPage,
				PerPage: gilabAPIMaxListSize,
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "can not list branches")
		}

		currentPage++

		if len(gitBranches) == 0 {
			break
		}

		for _, gitBranch := range gitBranches {
			lastCommitHoursAgo := time.Since(*gitBranch.Commit.CommittedDate).Hours()
			branchSlug := utils.GitlabSluglify(gitBranch.Name)

			if lastCommitHoursAgo > hoursInDay*staleBranchDays {
				result[branchSlug] = true
			} else {
				result[branchSlug] = false
			}
		}
	}

	return result, nil
}
