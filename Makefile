BUILD_CMD := go build -o music-release-publisher cmd/bot/main.go

.DEFAULT_GOAL := build

.PHONY: build
build: download/deps
	$(BUILD_CMD)

.PHONY: install
install:
	go install cmd/bot/main.go

.PHONY: download/deps
download/deps:
	go mod download

.PHONY: clean
clean:
	rm -f coverage.out
	go clean -testcache

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go tool oss.indeed.com/go/go-opine test -coverprofile=coverage.out -min-coverage 5

.PHONY: test/console
test/console: test
	@./scripts/cover.sh "func"

.PHONY: test/html
test/html: test
	@./scripts/cover.sh "html"

.PHONY: run
run: build
	./music-release-publisher $(ARGS)

.PHONY: run/debug
run/debug:
	go tool github.com/air-verse/air --build.cmd "$(BUILD_CMD)" --build.bin "./music-release-publisher" --build.args_bin "$(ARGS)"

.PHONY: run/quick
run/quick:
	go run -race cmd/bot/main.go $(ARGS)

.PHONY: update
update:
	go get -u ./...

.PHONY: sync/deps
sync/deps:
	go mod tidy

.PHONY: vet
vet:
	go vet ./...

.PHONY: errcheck
errcheck:
	go tool github.com/kisielk/errcheck ./...

.PHONY: staticcheck
staticcheck:
	go tool honnef.co/go/tools/cmd/staticcheck ./...

.PHONY: outdated
outdated:
	@./scripts/outdated.sh

.PHONY: refresh
refresh: update sync/deps download/deps
