all: clean build

clean:
	rm -rf dist

build:
	tsc
