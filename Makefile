# Build the dashboard, then the binary that embeds it.
build:
	cd web && bun install && bun run build
	go build -o bin/gocron ./cmd/gocron

# The default docker driver cannot build multi-platform images; create a docker-container builder once.
builder:
	docker buildx inspect multiarch >/dev/null 2>&1 || docker buildx create --name multiarch --driver docker-container

login:
	docker login

push: builder
	docker buildx build --builder multiarch \
		--platform linux/amd64,linux/arm64 \
		-t prongbang/gocron:$(tag) \
		-f deployments/Dockerfile \
		--push .

# make push_image version=1.1.0 -> pushes :latest and :1.1.0 for amd64 + arm64
push_image: builder
	docker buildx build --builder multiarch \
		--platform linux/amd64,linux/arm64 \
		-t prongbang/gocron:latest \
		-t prongbang/gocron:$(version) \
		-f deployments/Dockerfile \
		--push .
