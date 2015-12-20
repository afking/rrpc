
PROTOBUF = $(GOPATH)/src/github.com/golang/protobuf
LINK_RRPC_FILE = $(PROTOBUF)/protoc-gen-go/link_rrpc.go
all: install

install:
ifeq ("$(wildcard $(PROTOBUF)/.*)","")
	echo "protobuf doesn't exist"
	go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
endif

ifeq ("$(wildcard $(LINK_RRPC_FILE))","")
	touch $(LINK_RRPC_FILE)
	echo package main >> $(LINK_RRPC_FILE)
	echo >> $(LINK_RRPC_FILE)
	echo import _ \"github.com/afking/rrpc/plugin\" >> $(LINK_RRPC_FILE)
endif

	echo $(PROTOBUF) 
	$(MAKE) -C $(PROTOBUF)

clean:
	rm $(LINK_RRPC_FILE)
	$(MAKE) -C $(PROTOBUF)