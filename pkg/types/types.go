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
package types

type TagType string

func (t TagType) String() string {
	return string(t)
}

const (
	Unknown                 TagType = "Unknown"
	BranchNotFound          TagType = "BranchNotFound"
	ReleaseTagCanNotDelete  TagType = "ReleaseTagCanNotDelete"
	ReleaseTag              TagType = "ReleaseTag"
	SystemTag               TagType = "SystemTag"
	BranchStale             TagType = "BranchStale"
	BranchNotStaled         TagType = "BranchNotStaled"
	SnapshotTagCanNotDelete TagType = "SnapshotTagCanNotDelete"
	SnapshotStaled          TagType = "SnapshotStaled"
)

type DeleteTagInput struct {
	Repository string
	Tag        string
	TagType    TagType
}
type Provider interface {
	// Initialize provider
	Init(dryRun bool) error
	// List repositories in provider
	Repositories(filter string) ([]string, error)
	// List tags in provider
	Tags(repository string) ([]string, error)
	// Delete tag
	DeleteTag(deleteTag DeleteTagInput) error
	// Run post commands in provider
	PostCommand() error
}
