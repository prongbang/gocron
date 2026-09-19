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

push_image:
	make push tag=latest
	make push tag=1.0.4