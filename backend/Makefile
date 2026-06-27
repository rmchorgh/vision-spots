.PHONY: run test docker-build clean

run:
	@set -a && . ./.env && set +a && go run .

test:
	go test ./... -v

docker-build:
	docker build -t vision-spots-backend .

clean:
	rm -rf bin/
