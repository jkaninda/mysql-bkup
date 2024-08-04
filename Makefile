BINARY_NAME=mysql-bkup
include .env
export
run:
	go run . backup

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
	docker run --rm --network web --name mysql-bkup -v "./backup:/backup" -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" jkaninda/mysql-bkup bkup backup --prune --keep-last 2
docker-restore: docker-build
	docker run --rm --network web --name mysql-bkup -v "./backup:/backup" -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}"  -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" jkaninda/mysql-bkup  bkup restore -f ${FILE_NAME}


docker-run-scheduled: docker-build
	docker run --rm --network web --name mysql-bkup -v "./backup:/backup" -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" jkaninda/mysql-bkup  bkup backup --mode scheduled --period "* * * * *"


docker-run-scheduled-s3: docker-build
	docker run --rm --network web --name mysql-bkup -v "./backup:/backup" -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "ACCESS_KEY=${ACCESS_KEY}" -e "SECRET_KEY=${SECRET_KEY}" -e "BUCKET_NAME=${BUCKET_NAME}" -e "S3_ENDPOINT=${AWS_S3_ENDPOINT}" -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" jkaninda/mysql-bkup  bkup backup --storage s3 --mode scheduled --path /custom-path --period "* * * * *"

docker-run-s3: docker-build
	docker run --rm --network web --name mysql-bkup -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "ACCESS_KEY=${ACCESS_KEY}" -e "SECRET_KEY=${SECRET_KEY}" -e "AWS_S3_BUCKET_NAME=${AWS_S3_BUCKET_NAME}" -e "AWS_S3_ENDPOINT=${AWS_S3_ENDPOINT}" -e "AWS_REGION=eu2" -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" jkaninda/mysql-bkup  bkup backup --storage s3  --path /custom-path


docker-restore-s3: docker-build
	docker run --rm --network web --name mysql-bkup -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "ACCESS_KEY=${ACCESS_KEY}" -e "SECRET_KEY=${SECRET_KEY}" -e "BUCKET_NAME=${AWS_S3_BUCKET_NAME}" -e "S3_ENDPOINT=${AWS_S3_ENDPOINT}" -e "AWS_REGION=eu2" -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" jkaninda/mysql-bkup  bkup restore --storage s3 -f ${FILE_NAME} --path /custom-path

docker-run-ssh: docker-build
	docker run --rm --network web --name mysql-bkup -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "SSH_USER=${SSH_USER}" -e "SSH_HOST_NAME=${SSH_HOST_NAME}" -e "SSH_REMOTE_PATH=${SSH_REMOTE_PATH}" -e "SSH_PASSWORD=${SSH_PASSWORD}" -e "SSH_PORT=${SSH_PORT}" -e "SSH_IDENTIFY_FILE=${SSH_IDENTIFY_FILE}" -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" jkaninda/mysql-bkup  bkup backup --storage ssh

docker-restore-ssh: docker-build
	docker run --rm --network web --name mysql-bkup  -e "DB_HOST=${DB_HOST}" -e "DB_NAME=${DB_NAME}" -e "DB_USERNAME=${DB_USERNAME}" -e "DB_PASSWORD=${DB_PASSWORD}" -e "SSH_USER=${SSH_USER}" -e "SSH_HOST_NAME=${SSH_HOST_NAME}" -e "SSH_REMOTE_PATH=${SSH_REMOTE_PATH}" -e "SSH_PASSWORD=${SSH_PASSWORD}" -e "SSH_PORT=${SSH_PORT}" -e "GPG_PASSPHRASE=${GPG_PASSPHRASE}" -e "SSH_IDENTIFY_FILE=${SSH_IDENTIFY_FILE}" jkaninda/mysql-bkup  bkup restore --storage ssh -f ${FILE_NAME}

run-docs:
	cd docs && bundle exec jekyll serve -H 0.0.0.0 -t