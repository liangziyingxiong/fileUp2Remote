GOPATH=$(shell pwd)
BIN=filefetch

all:
	@echo "begin build $(BIN) ..."
	@mkdir -p $(GOPATH)/bin
	@cd $(GOPATH)/bin && export GOPATH=$(GOPATH) && go build $(BIN)

clean:
	@cd $(GOPATH)/bin && export GOPATH=$(GOPATH) && go clean $(BIN)
