CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-extldflags=-static -s -w"