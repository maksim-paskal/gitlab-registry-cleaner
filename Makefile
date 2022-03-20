tag=dev
image=paskalmaksim/gitlab-registry-cleaner:dev

test:
	./scripts/validate-license.sh
	go fmt ./cmd/... ./pkg/... ./internal/...
	go vet ./cmd/... ./pkg/... ./internal/...
	go test -race ./cmd/... ./pkg/... ./internal/...
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v

build:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --rm-dist --snapshot --skip-validate
	mv ./dist/gitlab-registry-cleaner_linux_amd64/gitlab-registry-cleaner ./gitlab-registry-cleaner
	docker build --pull . -t $(image)

push:
	docker push $(image)

scan:
	@trivy image \
	-ignore-unfixed --no-progress --severity HIGH,CRITICAL \
	$(image)
