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
package types_test

import (
	"testing"

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
)

func TestTagType(t *testing.T) {
	t.Parallel()

	tests := make(map[types.TagType]string)

	tests[types.Unknown] = "Unknown"
	tests[types.BranchNotFound] = "BranchNotFound"
	tests[types.ReleaseTagCanNotDelete] = "ReleaseTagCanNotDelete"
	tests[types.ReleaseTag] = "ReleaseTag"
	tests[types.SystemTag] = "SystemTag"
	tests[types.BranchStale] = "BranchStale"
	tests[types.BranchNotStaled] = "BranchNotStaled"
	tests[types.SnapshotTagCanNotDelete] = "SnapshotTagCanNotDelete"
	tests[types.SnapshotStaled] = "SnapshotStaled"

	for in, out := range tests {
		result := in.String()
		if result != out {
			t.Fatalf("result %s need %s", result, out)
		}
	}
}
