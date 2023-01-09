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
package api_test

import (
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/api"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
)

func newReleaseInput(tags map[string]types.TagType) *api.GetNotDeletableTagsInput {
	return &api.GetNotDeletableTagsInput{
		Tags:             tags,
		DateRegexp:       regexp.MustCompile(`^release-(\d{8}).*$`),
		NotDeleteDays:    10,
		MinNotDeleteTags: 3,
	}
}

func newSnapshotnput(tags map[string]types.TagType) *api.GetNotDeletableTagsInput {
	return &api.GetNotDeletableTagsInput{
		Tags:             tags,
		DateRegexp:       regexp.MustCompile(`^(\d{8})-snap$`),
		NotDeleteDays:    10,
		MinNotDeleteTags: 3,
	}
}

func TestGetGitlabProjectPath(t *testing.T) {
	t.Parallel()

	tests := make(map[string]string)

	tests["test/test/test/test"] = "test/test/test"
	tests["test/test/test"] = "test/test"
	tests["#@/#@/&&"] = "#@/#@"

	for in, out := range tests {
		result, err := api.GetGitlabProjectPath(in)
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
		_, err := api.GetGitlabProjectPath(test)
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

	result := api.GetNotDeletableTags(newReleaseInput(tags))

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
		t.Fatalf("tags not equals \n(%v)<=result\n(%v)<=need", result, need)
	}
}

func TestGetNotDeletableSnapshotTags(t *testing.T) {
	t.Parallel()

	tags := make(map[string]types.TagType)

	tags["20210316-snap"] = types.Unknown
	tags["20210421-snap"] = types.Unknown
	tags["20210504-snap"] = types.Unknown
	tags["20211117-snap"] = types.Unknown
	tags["20220331-snap"] = types.Unknown
	tags["20220415-snap"] = types.Unknown
	tags["20220606-snap"] = types.Unknown
	tags["20220615-snap"] = types.Unknown
	tags["20220809-snap"] = types.Unknown
	tags["20228899-snap"] = types.Unknown // fake date
	tags["90228809-snap"] = types.Unknown // date in future

	result := api.GetNotDeletableTags(newSnapshotnput(tags))

	need := []string{
		"20220809-snap",
		"20220615-snap",
		"20220606-snap",
	}

	sort.Strings(need)
	sort.Strings(result)

	if !reflect.DeepEqual(result, need) {
		t.Fatalf("tags not equals \n(%v)<=result\n(%v)<=need", result, need)
	}
}

func TestGetNotDeletableReleaseTagsRare(t *testing.T) {
	t.Parallel()

	tags := make(map[string]types.TagType)

	tags["release-20220320"] = types.ReleaseTag
	tags["release-20220320-amd64"] = types.ReleaseTag
	tags["release-20220320-arm64"] = types.ReleaseTag
	tags["release-20220219"] = types.ReleaseTag
	tags["release-20220219-amd64"] = types.ReleaseTag
	tags["release-20220219-arm64"] = types.ReleaseTag
	tags["release-20220118"] = types.ReleaseTag
	tags["release-20220118-amd64"] = types.ReleaseTag
	tags["release-20220118-arm64"] = types.ReleaseTag
	tags["release-20211217"] = types.ReleaseTag
	tags["release-20211217-amd64"] = types.ReleaseTag
	tags["release-20211217-arm64"] = types.ReleaseTag

	result := api.GetNotDeletableTags(newReleaseInput(tags))

	need := []string{
		"release-20220320",
		"release-20220320-amd64",
		"release-20220320-arm64",
		"release-20220219",
		"release-20220219-amd64",
		"release-20220219-arm64",
		"release-20220118",
		"release-20220118-amd64",
		"release-20220118-arm64",
	}

	sort.Strings(need)
	sort.Strings(result)

	if !reflect.DeepEqual(result, need) {
		t.Fatalf("tags not equals \n(%v)<=result\n(%v)<=need", result, need)
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if version := api.GetVersion(); version != "dev" {
		t.Fatalf("version %s is incorrect", version)
	}
}

func TestGetTagWithoutArch(t *testing.T) {
	t.Parallel()

	tests := make(map[string]string)

	tests["test1"] = "test1"
	tests["test2-arm64"] = "test2"
	tests["test3-amd64"] = "test3"

	for in, out := range tests {
		result := api.GetTagWithoutArch(in)
		if result != out {
			t.Fatalf("result %s need %s", result, out)
		}
	}
}

func TestCheckReleaseTag(t *testing.T) { //nolint:funlen
	t.Parallel()

	releaseTagRegexp := regexp.MustCompile(`^release-(\d{8}).*$`)

	type Test struct {
		Tag             string
		CommitTimestamp string
	}

	tests := make([]Test, 0)

	tests = append(tests, Test{
		Tag:             "release-20220325",
		CommitTimestamp: "2022-03-21T00:00:00+00:00",
	})

	tests = append(tests, Test{
		Tag:             "release-20220320",
		CommitTimestamp: "2022-03-15T00:00:00+00:00",
	})

	tests = append(tests, Test{
		Tag:             "release-20220301",
		CommitTimestamp: "2022-03-03T00:00:00+00:00",
	})

	for _, test := range tests {
		err := api.CheckReleaseTag(releaseTagRegexp, test.Tag, test.CommitTimestamp)
		if err != nil {
			t.Fatal(err)
		}
	}

	testsFailed := make([]Test, 0)

	// release bigger than 5 days
	testsFailed = append(testsFailed, Test{
		Tag:             "release-20220325",
		CommitTimestamp: "2022-03-31T00:00:00+00:00",
	})

	// release less than 5 days
	testsFailed = append(testsFailed, Test{
		Tag:             "release-20220320",
		CommitTimestamp: "2022-03-14T00:00:00+00:00",
	})

	// tag not valid
	testsFailed = append(testsFailed, Test{
		Tag:             "drelease-20220111",
		CommitTimestamp: "2022-03-15T00:00:00+00:00",
	})

	// tag date not valid
	testsFailed = append(testsFailed, Test{
		Tag:             "release-20220144",
		CommitTimestamp: "2022-03-15T00:00:00+00:00",
	})

	// release date in future
	testsFailed = append(testsFailed, Test{
		Tag:             "release-20990320",
		CommitTimestamp: "2099-03-19T00:00:00+00:00",
	})

	for _, test := range testsFailed {
		err := api.CheckReleaseTag(releaseTagRegexp, test.Tag, test.CommitTimestamp)
		if err == nil {
			t.Fatal("error must be " + test.Tag)
		}
	}
}

func TestGetFormattedTags(t *testing.T) {
	t.Parallel()

	tags := []string{
		"main",
		"release-20221226-1-master",
		"release-20221226-1-master-amd64",
		"release-20221226-1-master-arm64",
		"release-20230106-master",
		"release-20230106-master-amd64",
		"release-20230106-master-arm64",
	}

	result := api.GetTagsWithoutArch(tags)

	need := []string{
		"release-20221226-1-master",
		"release-20230106-master",
		"main",
	}

	sort.Strings(need)
	sort.Strings(result)

	if !reflect.DeepEqual(result, need) {
		t.Fatalf("tags not equals \n(%v)<=result\n(%v)<=need", result, need)
	}
}
