tag=dev
image=paskalmaksim/gitlab-registry-cleaner:$(tag)

test:
	./scripts/validate-license.sh
	go fmt ./cmd/... ./pkg/... ./internal/...
	go vet ./cmd/... ./pkg/... ./internal/...
	go test -race -coverprofile coverage.out ./cmd/... ./pkg/... ./internal/...
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v

coverage:
	go tool cover -html=coverage.out

build:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --rm-dist --snapshot --skip-validate
	mv ./dist/gitlab-registry-cleaner_linux_amd64_v1/gitlab-registry-cleaner ./gitlab-registry-cleaner
	docker build --pull . -t $(image)

push:
	docker push $(image)

scan:
	@trivy image \
	-ignore-unfixed --no-progress --severity HIGH,CRITICAL \
	$(image)
