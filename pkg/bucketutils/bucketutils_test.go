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
package bucketutils_test

import (
	"fmt"
	"testing"

	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/bucketutils"
)

func TestUtils(t *testing.T) {
	t.Parallel()

	items := make([]*bucketutils.BucketItem, 0)

	for i := 0; i < 10; i++ {
		items = append(items, &bucketutils.BucketItem{
			Value: fmt.Sprintf("test-%d", i),
		})
	}

	utils, err := bucketutils.NewBucketUtils("(?P<group>test)-(?P<id>[0-9]+)", items)
	if err != nil {
		t.Fatal(err)
	}

	getLastItemsGroupByID, err := utils.GetLastItemsGroupByID("test", 2)
	if err != nil {
		t.Error(err)
	}

	for _, item := range getLastItemsGroupByID {
		t.Log(item.Value)
	}

	// must return 8 items, because 2 items must leave
	if len(getLastItemsGroupByID) != 8 {
		t.Fatal("must return 8 items")
	}

	for _, item := range getLastItemsGroupByID {
		if item.Value == "test-9" || item.Value == "test-8" {
			t.Fatal("must not contain test-9 or test-8")
		}
	}
}
