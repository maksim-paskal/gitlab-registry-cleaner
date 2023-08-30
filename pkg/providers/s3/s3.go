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
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/types"
	"github.com/maksim-paskal/gitlab-registry-cleaner/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	s3Region         = flag.String("s3.region", os.Getenv("S3_REGION"), "")
	s3Accesskey      = flag.String("s3.accesskey", os.Getenv("S3_ACCESSKEY"), "")
	s3Secretkey      = flag.String("s3.secretkey", os.Getenv("S3_SECRETKEY"), "")
	s3Bucket         = flag.String("s3.bucket", os.Getenv("S3_BUCKET"), "")
	s3Endpoint       = flag.String("s3.endpoint", os.Getenv("S3_ENDPOINT"), "")
	s3DisableSSL     = flag.String("s3.disable-ssl", os.Getenv("S3_DISABLE_SSL"), "")
	s3ForcePathStyle = flag.String("s3.force-path-style", os.Getenv("S3_FORCE_PATH_STYLE"), "")
	registryFolder   = flag.String("s3.registry-folder", "docker/registry/v2/repositories/", "")
)

const listMax = 1000

type Provider struct {
	dryRun        bool
	svc           *s3.S3
	repositories  map[string]bool
	deletefolders map[string]bool
}

func (p *Provider) deleteBucketFolder(ctx context.Context, directory string) error {
	iter := s3manager.NewDeleteListIterator(p.svc, &s3.ListObjectsInput{
		Bucket: aws.String(*s3Bucket),
		Prefix: aws.String(directory),
	})

	if err := s3manager.NewBatchDeleteWithClient(p.svc).Delete(ctx, iter); err != nil {
		return errors.Wrap(err, "failed to delete files under given directory")
	}

	return nil
}

func (p *Provider) listRepository(ctx context.Context, folder string) error {
	resp, err := p.svc.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
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

		if err = p.listRepository(ctx, *item.Prefix); err != nil {
			return errors.Wrap(err, "failed to list objects")
		}
	}

	return nil
}

func (p *Provider) Init(_ context.Context, dryRun bool) error {
	p.dryRun = dryRun

	config := aws.Config{}

	if len(*s3Accesskey) > 0 && len(*s3Secretkey) > 0 {
		config.Credentials = credentials.NewStaticCredentials(*s3Accesskey, *s3Secretkey, "")
	}

	if len(*s3Region) > 0 {
		config.Region = aws.String(*s3Region)
	}

	if len(*s3Endpoint) > 0 {
		config.Endpoint = aws.String(*s3Endpoint)
	}

	if *s3DisableSSL == "true" {
		config.DisableSSL = aws.Bool(true)
	}

	if *s3ForcePathStyle == "true" {
		config.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession()
	if err != nil {
		return errors.Wrap(err, "failed to create aws session")
	}

	p.svc = s3.New(sess, &config)

	p.deletefolders = make(map[string]bool)

	return nil
}

func (p *Provider) Repositories(ctx context.Context, filter string) ([]string, error) {
	p.repositories = make(map[string]bool)

	if err := p.listRepository(ctx, *registryFolder); err != nil {
		return nil, errors.Wrap(err, "failed to list objects")
	}

	repositories := make([]string, 0)

	for repo := range p.repositories {
		repositories = append(repositories, repo)
	}

	if len(filter) > 0 {
		return utils.FilterStrings(repositories, filter), nil
	}

	return repositories, nil
}

func (p *Provider) Tags(ctx context.Context, repository string) ([]string, error) {
	tagsFolder := fmt.Sprintf("%s%s/_manifests/tags/", *registryFolder, repository)

	resp, err := p.svc.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
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

func (p *Provider) DeleteTag(_ context.Context, deleteTag types.DeleteTagInput) error {
	p.deletefolders[fmt.Sprintf("%s%s/_manifests/tags/%s/", *registryFolder, deleteTag.Repository, deleteTag.Tag)] = true

	return nil
}

func (p *Provider) PostCommand(ctx context.Context) error {
	for repo := range p.deletefolders {
		if p.dryRun {
			log.Warnf("delete folder %s ", repo)
		} else {
			if err := p.deleteBucketFolder(ctx, repo); err != nil {
				return errors.Wrapf(err, "failed to delete folder %s", repo)
			}
		}
	}

	log.Infof("Done")

	return nil
}
