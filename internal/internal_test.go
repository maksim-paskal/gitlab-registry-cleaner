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
package internal_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/paskal-maksim/gitlab-registry-cleaner/internal"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
)

func TestGetGitlabProjectPath(t *testing.T) {
	t.Parallel()

	tests := make(map[string]string)

	tests["test/test/test/test"] = "test/test/test"
	tests["test/test/test"] = "test/test"
	tests["#@/#@/&&"] = "#@/#@"

	for in, out := range tests {
		result, err := internal.GetGitlabProjectPath(in)
		if err != nil {
			t.Fatal(err)
		}

		if result != out {
			t.Fatalf("result %s need %s", result, out)
		}
	}

	testsToFail := []string{
		"test",
		"test/test",
		"test/test:test",
	}

	for _, test := range testsToFail {
		_, err := internal.GetGitlabProjectPath(test)
		if err == nil {
			t.Fatal("must throw error")
		}
	}
}

func TestGetNotDeletableReleaseTagsFrequent(t *testing.T) {
	t.Parallel()

	tags := make(map[string]types.TagType)

	tags["release-20220320"] = types.ReleaseTag
	tags["release-20220319"] = types.ReleaseTag
	tags["release-20220319-patch1"] = types.ReleaseTag
	tags["release-20220319-patch2"] = types.ReleaseTag
	tags["release-20220318-test"] = types.ReleaseTag
	tags["release-20220311"] = types.ReleaseTag
	tags["release-20220310"] = types.ReleaseTag
	tags["release-20220301"] = types.ReleaseTag
	tags["release-20220228"] = types.ReleaseTag
	tags["release-20220199"] = types.ReleaseTag     // fake tag
	tags["release-99990101"] = types.ReleaseTag     // fake tag
	tags["test-branch"] = types.ReleaseTag          // fake tag
	tags["test-20220318-branch"] = types.ReleaseTag // fake tag

	result := internal.GetNotDeletableReleaseTags(tags)

	need := []string{
		"release-20220320",
		"release-20220319",
		"release-20220319-patch1",
		"release-20220319-patch2",
		"release-20220318-test",
		"release-20220311",
	}

	sort.Strings(need)
	sort.Strings(result)

	if !reflect.DeepEqual(result, need) {
		t.Fatalf("tags not equals result=%v", result)
	}
}

func TestGetNotDeletableReleaseTagsRare(t *testing.T) {
	t.Parallel()

	tags := make(map[string]types.TagType)

	tags["release-20220320"] = types.ReleaseTag
	tags["release-20220219"] = types.ReleaseTag
	tags["release-20220118"] = types.ReleaseTag
	tags["release-20211217"] = types.ReleaseTag

	result := internal.GetNotDeletableReleaseTags(tags)

	need := []string{
		"release-20220320",
		"release-20220219",
		"release-20220118",
	}

	sort.Strings(need)
	sort.Strings(result)

	if !reflect.DeepEqual(result, need) {
		t.Fatalf("tags not equals result=%v", result)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if version := internal.GetVersion(); version != "dev" {
		t.Fatalf("version %s is incorrect", version)
	}
}
