run:
	(fd -e go; echo .env) | entr -r sh -c 'clear && go run .'

build:
	go build -o bin/afloat

build-run:
	(fd -e go; echo .env) | entr -r sh -c 'clear && go build -o bin/afloat && ./bin/afloat'

clean:
	rm -rf ./bin
