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
package utils_test

import (
	"testing"

	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/utils"
)

func TestGitlabSluglify(t *testing.T) {
	t.Parallel()

	tests := make(map[string]string)

	tests["$test/test/test/test$"] = "test-test-test-test"
	tests["@test/test/test@"] = "test-test-test"
	tests["#test#"] = "test"
	tests["1234567890123456789012345678901234567890123456789012345678901234567890"] = "123456789012345678901234567890123456789012345678901234567890123" //nolint: lll

	for in, out := range tests {
		result := utils.GitlabSluglify(in)

		if result != out {
			t.Fatalf("result %s need %s", result, out)
		}
	}
}

func TestStringInSlice(t *testing.T) {
	t.Parallel()

	if !utils.StringInSlice("test", []string{"test"}) {
		t.Fatal("String must contains in array")
	}

	if utils.StringInSlice("test1", []string{"test"}) {
		t.Fatal("String must contains in array")
	}
}
