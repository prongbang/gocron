# make push tag=1.0.0
push:
	docker build -t prongbang/gocron:$(tag) -f deployments/Dockerfile .
	docker tag prongbang/gocron:$(tag) prongbang/gocron:$(tag)
	docker push prongbang/gocron:$(tag)