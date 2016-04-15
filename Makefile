DOCKER_IMAGE = lightcode/kube2consul

build:
	export GO15VENDOREXPERIMENT=1
	go build -v -i

install:
	go install -v

docker:
	docker build -t $(DOCKER_IMAGE) .
