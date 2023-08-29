tag=dev
image=paskalmaksim/gitlab-registry-cleaner:$(tag)

test:
	./scripts/validate-license.sh
	go fmt ./cmd/... ./pkg/... ./internal/...
	go vet ./cmd/... ./pkg/... ./internal/...
	go test -race -coverprofile coverage.out ./cmd/... ./pkg/... ./internal/...
	# test ci check on valid tag
	CI_COMMIT_REF_NAME=release-20230515-test \
	CI_COMMIT_TIMESTAMP="2023-05-12T08:56:11Z" \
	go run ./cmd/main/main.go -ci.check
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v

coverage:
	go tool cover -html=coverage.out

lint:
	ct lint --all

build:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --clean --snapshot --skip-validate
	mv ./dist/gitlab-registry-cleaner_linux_amd64_v1/gitlab-registry-cleaner ./gitlab-registry-cleaner
	docker build --pull --push . -t $(image)

push:
	docker push $(image)

scan:
	@trivy image \
	-ignore-unfixed --no-progress --severity HIGH,CRITICAL \
	$(image)
