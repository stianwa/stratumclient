GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get -u

all:  lint test


test:
	@$(GOCMD) test -v

lint:
	@go vet *.go
	@go fmt *.go
	@misspell .
	@golint *.go
	@gocyclo -top 4 .
	@gocyclo -over 50 .
	@ineffassign .
