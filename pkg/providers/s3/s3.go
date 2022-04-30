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
package s3

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/paskal-maksim/gitlab-registry-cleaner/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	s3Region       = flag.String("s3.region", os.Getenv("S3_REGION"), "")
	s3Accesskey    = flag.String("s3.accesskey", os.Getenv("S3_ACCESSKEY"), "")
	s3Secretkey    = flag.String("s3.secretkey", os.Getenv("S3_SECRETKEY"), "")
	s3Bucket       = flag.String("s3.bucket", os.Getenv("S3_BUCKET"), "")
	registryFolder = flag.String("s3.registry-folder", "docker/registry/v2/repositories/", "")
)

const listMax = 1000

type Provider struct {
	dryRun        bool
	svc           *s3.S3
	repositories  map[string]bool
	deletefolders map[string]bool
}

func (p *Provider) deleteBucketFolder(directory string) error {
	ctx := context.Background()

	iter := s3manager.NewDeleteListIterator(p.svc, &s3.ListObjectsInput{
		Bucket: aws.String(*s3Bucket),
		Prefix: aws.String(directory),
	})

	if err := s3manager.NewBatchDeleteWithClient(p.svc).Delete(ctx, iter); err != nil {
		return errors.Wrap(err, "failed to delete files under given directory")
	}

	return nil
}

func (p *Provider) listRepository(folder string) error {
	resp, err := p.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:    aws.String(*s3Bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(folder),
		MaxKeys:   aws.Int64(listMax),
	})
	if err != nil {
		return errors.Wrap(err, "failed to list objects")
	}

	for _, item := range resp.CommonPrefixes {
		if strings.HasSuffix(*item.Prefix, "/_uploads/") {
			// remove temporary folder
			p.deletefolders[fmt.Sprintf("%s%s", *registryFolder, *item.Prefix)] = true
		}

		if strings.Contains(*item.Prefix, "/_layers/") || strings.Contains(*item.Prefix, "/_manifests/") || strings.Contains(*item.Prefix, "/_uploads/") { //nolint:lll
			repository := *item.Prefix
			repository = strings.TrimSuffix(repository, "/_layers/")
			repository = strings.TrimSuffix(repository, "/_manifests/")
			repository = strings.TrimSuffix(repository, "/_uploads/")
			repository = strings.TrimSuffix(repository, "/")
			repository = strings.TrimPrefix(repository, *registryFolder)

			p.repositories[repository] = true

			continue
		}

		if err = p.listRepository(*item.Prefix); err != nil {
			return errors.Wrap(err, "failed to list objects")
		}
	}

	return nil
}

func (p *Provider) Init(dryRun bool) error {
	p.dryRun = dryRun

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(*s3Accesskey, *s3Secretkey, ""),
		Region:      aws.String(*s3Region),
	})
	if err != nil {
		return errors.Wrap(err, "failed to create aws session")
	}

	p.svc = s3.New(sess)

	p.deletefolders = make(map[string]bool)

	return nil
}

func (p *Provider) Repositories() ([]string, error) {
	p.repositories = make(map[string]bool)

	if err := p.listRepository(*registryFolder); err != nil {
		return nil, errors.Wrap(err, "failed to list objects")
	}

	repositories := make([]string, 0)

	for repo := range p.repositories {
		repositories = append(repositories, repo)
	}

	return repositories, nil
}

func (p *Provider) Tags(repository string) ([]string, error) {
	tagsFolder := fmt.Sprintf("%s%s/_manifests/tags/", *registryFolder, repository)

	resp, err := p.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:    aws.String(*s3Bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(tagsFolder),
		MaxKeys:   aws.Int64(listMax),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list objects")
	}

	tags := make([]string, 0)

	for _, item := range resp.CommonPrefixes {
		tag := *item.Prefix
		tag = strings.TrimPrefix(tag, *registryFolder)
		tag = strings.TrimPrefix(tag, repository)
		tag = strings.TrimPrefix(tag, "/_manifests/tags/")
		tag = strings.TrimSuffix(tag, "/")

		tags = append(tags, tag)
	}

	if len(tags) == 0 {
		p.deletefolders[fmt.Sprintf("%s%s/", *registryFolder, repository)] = true

		log.Debugf("%s no tags found", repository)
	}

	return tags, nil
}

func (p *Provider) DeleteTag(repository string, tag string, tagType types.TagType) error {
	p.deletefolders[fmt.Sprintf("%s%s/_manifests/tags/%s/", *registryFolder, repository, tag)] = true

	return nil
}

func (p *Provider) PostCommand() error {
	for repo := range p.deletefolders {
		if p.dryRun {
			log.Warnf("delete folder %s ", repo)
		} else {
			if err := p.deleteBucketFolder(repo); err != nil {
				return errors.Wrapf(err, "failed to delete folder %s", repo)
			}
		}
	}

	log.Infof("Done")

	return nil
}
