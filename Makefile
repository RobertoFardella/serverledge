BIN=bin

all: serverledge executor serverledge-cli lb 

serverledge:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go 

lb:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/main.go

serverledge-cli:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/cli/main.go

executor:
	CGO_ENABLED=0 GOOS=linux go build -o $(BIN)/$@ cmd/$@/executor.go

DOCKERHUB_USER=grussorusso
images:  image-python310 image-nodejs17ng image-base
image-python310:
	docker build -t $(DOCKERHUB_USER)/serverledge-python310 -f images/python310/Dockerfile .
image-base:
	docker build -t $(DOCKERHUB_USER)/serverledge-base -f images/base-alpine/Dockerfile .
image-nodejs17ng:
	docker build -t $(DOCKERHUB_USER)/serverledge-nodejs17ng -f images/nodejs17ng/Dockerfile .

push-images:
	docker push $(DOCKERHUB_USER)/serverledge-python310
	docker push $(DOCKERHUB_USER)/serverledge-base
	docker push $(DOCKERHUB_USER)/serverledge-nodejs17ng

etcd_start:
	./scripts/start-etcd.sh

etcd_stop:
	./scripts/stop-etcd.sh

node_start:
	./bin/serverledge

test:
	go test -v ./...

.PHONY: serverledge serverledge-cli lb executor test images

	
#.PHONY: serverledge serverledge-cli lb executor test images
#Descrizione: Specifica che i target elencati non corrispondono a file reali nel filesystem. Questo garantisce che le regole associate vengano eseguite ogni volta che vengono invocate.

#$@ indichiamo il target corrente

#Descrizione:con il target test Esegue tutti i test Go presenti nel progetto in modalità verbosa.
