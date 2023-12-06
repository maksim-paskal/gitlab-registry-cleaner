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
package bucketutils

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type BucketItem struct {
	id    int
	group string
	Value string
}

func (b *BucketItem) String() string {
	return fmt.Sprintf("%s (%s-%d)", b.Value, b.group, b.id)
}

func NewBucketUtils(bucketRegexp string, items []*BucketItem) (*BucketUtils, error) {
	if len(bucketRegexp) == 0 {
		return nil, errors.New("bucketRegexp is empty")
	}

	re2, err := regexp.Compile(bucketRegexp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile regexp")
	}

	result := &BucketUtils{
		re:            re2,
		groupPosition: slices.Index(re2.SubexpNames(), "group"),
		idPosition:    slices.Index(re2.SubexpNames(), "id"),
		Items:         items,
	}

	if result.groupPosition < 0 || result.idPosition < 0 {
		return nil, errors.New("regexp must contain group and id")
	}

	return result, nil
}

type BucketUtils struct {
	Items         []*BucketItem
	re            *regexp.Regexp
	groupPosition int
	idPosition    int
}

func (b *BucketUtils) GetGroups() map[string]int {
	groups := make(map[string]int)

	for _, item := range b.Items {
		groups[b.findStringSubmatch(item.Value, b.groupPosition)]++
	}

	return groups
}

func (b *BucketUtils) findStringSubmatch(value string, pos int) string {
	result := b.re.FindStringSubmatch(value)
	if result == nil {
		return ""
	}

	if len(result) <= pos {
		return ""
	}

	return result[pos]
}

func (b *BucketUtils) GetGroupItems(group string) ([]*BucketItem, error) {
	var items []*BucketItem

	for _, item := range b.Items {
		if b.findStringSubmatch(item.Value, b.groupPosition) == group {
			groupItem := *item

			if idValue := b.findStringSubmatch(item.Value, b.idPosition); len(idValue) > 0 {
				id, err := strconv.Atoi(idValue)
				if err != nil {
					return nil, errors.Wrap(err, "failed to convert id to int")
				}

				groupItem.id = id
			}

			groupItem.group = group

			items = append(items, &groupItem)
		}
	}

	return items, nil
}

func (b *BucketUtils) GetLastItemsGroupByID(group string, limit int) ([]*BucketItem, error) {
	items, err := b.GetGroupItems(group)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get group items")
	}

	if len(items) <= limit {
		return make([]*BucketItem, 0), nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].id > items[j].id
	})

	for _, item := range items[:limit] {
		log.Infof("keeping %s", item.Value)
	}

	return items[limit:], nil
}
