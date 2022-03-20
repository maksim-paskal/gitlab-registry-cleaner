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

type TagType uint32

const (
	CanNotDelete TagType = iota
	BranchNotFound
	ReleaseTagCanNotDelete
	ReleaseTag
	SystemTag
	BranchStale
)

type Provider interface {
	// Initialize provider
	Init() error
	// List repositories in provider
	Repositories() ([]string, error)
	// List tags in provider
	Tags(repository string) ([]string, error)
	// Delete tag
	DeleteTag(repository string, tag string, tagType TagType) error
	// Run post commands in provider
	PostCommand() error
}
