test:
	@./hack/test.sh

build:
	@VERSION=${VERSION}./hack/build.sh

# lint check of the api endpoint
api:
	@exit 0

lint:
	@exit 0