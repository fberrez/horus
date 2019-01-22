BUILD_ARGS := -ldflags "-X github.com/fberrez/build.Version=0.0.3"
MAIN_LOCATION := ./cmd/horus
BINARY := output/horus

all:
	go build -o $(BINARY) $(BUILD_ARGS) $(MAIN_LOCATION)/*.go

clean:
	rm -rf $(BINARY)

test:
	go test -v -race -cover -bench=. -coverprofile=cover.profile ./...

fmt:
	for filename in $$(find . -path ./vendor -prune -o -name '*.go' -print); do \
		gofmt -w -l -s $$filename ;\
	done
