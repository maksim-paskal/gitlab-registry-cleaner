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
package bucketcleaner

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/bucketutils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type BucketCleaner struct {
	svc     *s3.S3
	listMax int64
	Pattern string
	MaxSize int
	DryRun  bool
}

func (b *BucketCleaner) Validate() error {
	if len(b.Pattern) == 0 {
		return errors.New("pattern is empty")
	}

	if b.MaxSize <= 0 {
		return errors.New("maxSize must be bigger than zero")
	}

	return nil
}

func (b *BucketCleaner) Init(awsConfig aws.Config) error {
	sess, err := session.NewSession()
	if err != nil {
		return errors.Wrap(err, "failed to create aws session")
	}

	b.svc = s3.New(sess, &awsConfig)

	return nil
}

func (b *BucketCleaner) deleteBucketFolder(ctx context.Context, bucket, folder string) error {
	log.Warnf("deleting %s/%s", bucket, folder)

	if b.DryRun {
		return nil
	}

	iter := s3manager.NewDeleteListIterator(b.svc, &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(folder),
	})

	if err := s3manager.NewBatchDeleteWithClient(b.svc).Delete(ctx, iter); err != nil {
		return errors.Wrap(err, "failed to delete files under given directory")
	}

	return nil
}

func (b *BucketCleaner) PurgeBucket(ctx context.Context, bucket, folder string) error {
	if b.DryRun {
		log.Warn("dry run enabled")
	}

	if len(bucket) == 0 {
		return errors.New("bucket is empty")
	}

	resp, err := b.svc.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(folder),
		MaxKeys:   aws.Int64(b.listMax),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list objects")
	}

	items := make([]*bucketutils.BucketItem, len(resp.CommonPrefixes))

	for i, c := range resp.CommonPrefixes {
		items[i] = &bucketutils.BucketItem{
			Value: *c.Prefix,
		}
	}

	bucketUtils, err := bucketutils.NewBucketUtils(b.Pattern, items)
	if err != nil {
		return errors.Wrap(err, "failed to create bucket utils")
	}

	for group := range bucketUtils.GetGroups() {
		items, err := bucketUtils.GetLastItemsGroupByID(group, b.MaxSize)
		if err != nil {
			return errors.Wrap(err, "failed to get group items")
		}

		for _, item := range items {
			if ctx.Err() != nil {
				return errors.Wrap(ctx.Err(), "context")
			}

			if err := b.deleteBucketFolder(ctx, bucket, item.Value); err != nil {
				return errors.Wrapf(err, "failed to delete bucket folder %s/%s", bucket, item.Value)
			}
		}
	}

	return nil
}

func NewBucketCleaner() *BucketCleaner {
	return &BucketCleaner{
		listMax: 1000, //nolint:gomnd
	}
}
