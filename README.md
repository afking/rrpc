rrpc
====

A rabbitmq rpc service.


Installation
------------

Install the plugin with 
```
cd plugin
make
```
ensure your GOPATH environment is set.

Running
-------

```
protoc --go_out=plugins=rrpc:. *.proto
```

TODO
----
everything
