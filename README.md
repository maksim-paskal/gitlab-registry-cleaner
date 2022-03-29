# Gitlab Docker Registry Cleaner

## Motivation

With DevOps practices, docker registry growing fast with developing new features. Need some tool to purge stale docker registry tags in docker repository.

# Installation

create custom [values.yaml](charts/gitlab-registry-cleaner/values.yaml) for example

```yaml
env:
- name: GITLAB_TOKEN
  value: sometoken
- name: GITLAB_URL
  value: https://git.lab/api/v4
- name: REMOTE_REGISTRY
  value: "https://someregistry.azurecr.io"
- name: REMOTE_REGISTRY_USER
  value: "someuser"
- name: REMOTE_REGISTRY_PASSWORD
  value: "sometoken"

args:
- -metrics.pushgateway=http://prometheus-pushgateway.prometheus.svc.cluster.local:9091

tolerations:
- key: "kubernetes.azure.com/scalesetpriority"
  operator: "Equal"
  value: "spot"
  effect: "NoSchedule"
```

```bash
helm repo add maksim-paskal-gitlab-registry-cleaner https://maksim-paskal.github.io/gitlab-registry-cleaner
helm repo update

helm upgrade gitlab-registry-cleaner \
--install \
--create-namespace \
--namespace gitlab-registry-cleaner \
maksim-paskal-gitlab-registry-cleaner/gitlab-registry-cleaner \
--values values.yaml
```

## Requirements

All docker registry artifacts must contains the path of Gitlab project and sluglify tag of git branch or git tag

In Gitlab CI/CD is very simple to make it with

```yaml
build:
  image: docker:dind
  script: |
    docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY

    export BUILD_STAGE_NAME=someimagename
    export BUILD_IMAGE_NAME=$CI_REGISTRY/$CI_PROJECT_PATH/$BUILD_STAGE_NAME:$CI_COMMIT_REF_SLUG

    docker build --pull -t $BUILD_IMAGE_NAME .
    docker push $BUILD_IMAGE_NAME
```

All git tags that use in stage/prod envieroment must be named with `release-YYYMMDD` where `YYYYMMDD` is release date

## Clearing docker registry tags

Clearing docker tags will be peformed with this logic

1. If docker registry have this release tags

```bash
release-20220320
release-20220319
release-20220319-patch1
release-20220319-patch2
release-20220318-test
release-20220311
release-20220310 # will be removed
release-20220301 # will be removed
release-20220221 # will be removed
```

`gitlab-registry-cleaner` will leave only last 10 day of release tags

2. If docker registry tag exists and there if no git tag (branch was merged to main branch) - docker tag will be removed

