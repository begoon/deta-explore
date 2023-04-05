EXE=exe

all: run

run:
	go run ./

build:
	CGO_ENABLED=0 go build -o $(EXE) ./

build-space:
	GOOS=linux GOARCH=amd64 go build -o micro/$(EXE) ./

deploy: build-space push

push:
	(cd micro && space push)
