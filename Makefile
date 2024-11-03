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
	make push tag=1.0.3