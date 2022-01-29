
# from https://github.com/restic/restic/tree/master/docker
#!/bin/sh

set -e

echo "Build binary using golang docker image"
docker run --rm -ti \
  -v $(pwd):/go/src/github.com/xtforgame/agak \
  -w /go/src/github.com/xtforgame/agak \
  -e CGO_ENABLED=1 \
  -e GOOS=linux \
  -e GO111MODULE=on \
  golang:1.12-alpine3.9 go build -mod=vendor -o ./build/alpine3.9/agak main/server/server.go

docker run --rm -ti \
  -v $(pwd):/go/src/github.com/xtforgame/agak \
  -w /go/src/github.com/xtforgame/agak \
  -e CGO_ENABLED=1 \
  -e GOOS=linux \
  -e GO111MODULE=on \
  golang:1.12-alpine3.9 go build -mod=vendor -o ./dist/alpine3.9/agak main/server/server.go

echo "Build docker image xtforgame/agak:0.1"
docker build --rm -t xtforgame/agak:0.1 -f docker/alpine3.9/Dockerfile .

# docker run --rm -ti \
#   -p 8081:8080 \
#   -v $(pwd)/runtime:/usr/agak/runtime \
#   -v $(pwd)/examples:/usr/agak/examples \
#   -w /usr/agak \
#   xtforgame/agak:0.1 agak
