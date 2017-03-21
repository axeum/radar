image_version=$1

echo "Compile and build the application"
docker run --rm -v $(pwd):/usr/src/myapp -w /usr/src/myapp -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.8 bash -c "go get -d -v; go build -a --installsuffix cgo -v -o radar"

echo "Building image radar:$image_version"
docker build --no-cache=true -t ictu/radar:$image_version .

echo "Pushing radar:$image_version and radar:latest"
docker push ictu/radar:$image_version

echo "Cleanup"
rm -f radar
