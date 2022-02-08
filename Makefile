GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get -u

all:  lint test


test:
	@$(GOCMD) test -v

lint:
	@misspell .
	@golint
	@gocyclo -top 4 .
	@gocyclo -over 50 .
	@ineffassign .
	@gofmt -d .
