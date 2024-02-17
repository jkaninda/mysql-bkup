BINARY_NAME=mysql-bkup
include .env
export
run:
	go run .

build:
	go build -o bin/${BINARY_NAME} .

compile:
	GOOS=darwin GOARCH=arm64 go build -o bin/${BINARY_NAME}-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}-darwin-amd64 .
	GOOS=linux GOARCH=arm64 go build -o bin/${BINARY_NAME}-linux-arm64 .
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-linux-amd64 .

docker-build:
	docker build -f docker/Dockerfile  -t jkaninda/mysql-bkup:latest .

docker-run: docker-build
	docker run --rm --network internal --privileged --device /dev/fuse --name mysql-bkup -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" jkaninda/mysql-bkup  bkup backup --prune --keep-last 2


docker-run-scheduled: docker-build
	docker run --rm --network internal --privileged --device /dev/fuse --name mysql-bkup -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -v "./backup:/backup"  jkaninda/mysql-bkup  bkup backup --prune --keep-last=2 --mode scheduled --period "* * * * *"


docker-run-scheduled-s3: docker-build
	docker run --rm --network internal --privileged --device /dev/fuse --name mysql-bkup -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "ACCESS_KEY=${ACCESS_KEY}" -e "SECRET_KEY=${SECRET_KEY}" -e "BUCKET_NAME=${BUCKET_NAME}" -e "S3_ENDPOINT=${S3_ENDPOINT}" jkaninda/mysql-bkup  bkup backup --storage s3 --mode scheduled --path /custom-path --period "* * * * *"

docker-restore-s3: docker-build
	docker run --rm --network internal --privileged --device /dev/fuse --name mysql-bkup -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "ACCESS_KEY=${ACCESS_KEY}" -e "SECRET_KEY=${SECRET_KEY}" -e "BUCKET_NAME=${BUCKET_NAME}" -e "S3_ENDPOINT=${S3_ENDPOINT}" -e "FILE_NAME=${FILE_NAME}" jkaninda/mysql-bkup  bkup restore --storage s3 --path /custom-path

