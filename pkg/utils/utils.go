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
package utils

import (
	"regexp"
	"strings"
)

// implement sluglify function in gitlab
// https://gitlab.com/gitlab-org/gitlab/-/blob/master/lib/gitlab/utils.rb#L92
func GitlabSluglify(text string) string {
	result := strings.ToLower(text)

	m := regexp.MustCompile(`[^a-z0-9]`)
	result = m.ReplaceAllString(result, "-")

	// result must not start or finish with dash
	result = strings.TrimPrefix(result, "-")
	result = strings.TrimSuffix(result, "-")

	return result
}
