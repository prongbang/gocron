# Build the dashboard, then the binary that embeds it.
build:
	cd web && bun install && bun run build
	go build -o bin/gocron ./cmd/gocron

login:
	docker login

push:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t prongbang/gocron:$(tag) \
		-f deployments/Dockerfile \
		--push .

# make push_image version=1.1.0 -> pushes :latest and :1.1.0 for amd64 + arm64
push_image:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t prongbang/gocron:latest \
		-t prongbang/gocron:$(version) \
		-f deployments/Dockerfile \
		--push .
