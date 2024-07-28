ifneq (,$(wildcard ./.env))
    include .env
    export
endif


run:
	go run ./app

clean:
	rm -rf ./bin/*