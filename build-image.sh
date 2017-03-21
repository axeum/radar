image_version=$1

echo "Compile the application"
docker run --rm -v $(pwd):/usr/src/myapp -w /usr/src/myapp -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 golang:1.6 bash -c "go get -d -v; go build -a --installsuffix cgo -v"

echo "Building radar:$image_version"
docker build --no-cache=true -t ictu/radar:$image_version .

echo "Pushing plumber:$image_version and plumber:latest"
#docker push ictu/radar:$image_version

echo "Cleanup"
rm -f radar
