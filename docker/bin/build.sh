# from https://github.com/restic/restic/tree/master/docker
#!/bin/sh

set -e

docker build -t agak-build:latest ./docker/bin

# https://medium.com/travis-on-docker/how-to-cross-compile-go-programs-using-docker-beaa102a316d
docker run --rm -it -v "$GOPATH":/go -w /go/src/github.com/xtforgame/agak agak-build:latest sh -c '
export GO111MODULE=on
for GOARCH in 386 amd64; do
  for GOOS in darwin linux windows freebsd; do
    echo "Building $GOOS-$GOARCH"
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    go build -mod=vendor -o build/bin/server-$GOOS-$GOARCH -a -ldflags "-extldflags -static" -tags netgo -installsuffix netgo main/server/server.go
    ldd build/bin/server-$GOOS-$GOARCH
  done
done
for GOARCH in arm ; do
  for GOOS in linux freebsd; do
    echo "Building $GOOS-$GOARCH"
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    go build -mod=vendor -o build/bin/server-$GOOS-$GOARCH -a -ldflags "-extldflags -static" -tags netgo -installsuffix netgo main/server/server.go
    ldd build/bin/server-$GOOS-$GOARCH
  done
done
for GOARCH in arm64 ; do
  for GOOS in linux; do
    echo "Building $GOOS-$GOARCH"
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    go build -mod=vendor -o build/bin/server-$GOOS-$GOARCH -a -ldflags "-extldflags -static" -tags netgo -installsuffix netgo main/server/server.go
    ldd build/bin/server-$GOOS-$GOARCH
  done
done
'
