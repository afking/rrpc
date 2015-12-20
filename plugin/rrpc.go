package rrpc

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath = "golang.org/x/net/context"
	rrpcPkgPath    = "github.com/afking/rrpc"
)

func init() {
	generator.RegisterPlugin(new(rrpc))
}

// rrpc implements rabbitmq binding for rpc services.
type rrpc struct {
	gen *generator.Generator
}

func (r *rrpc) Name() string {
	return "rrpc"
}

var (
	contextPkg string
	rrpcPkg    string
)

// Init initializes the plugin.
func (r *rrpc) Init(gen *generator.Generator) {
	r.gen = gen
	contextPkg = generator.RegisterUniquePackageName("context", nil)
	rrpcPkg = generator.RegisterUniquePackageName("rrpc", nil)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (r *rrpc) objectNamed(name string) generator.Object {
	r.gen.RecordTypeUse(name)
	return r.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (r *rrpc) typeName(str string) string {
	return r.gen.TypeName(r.objectNamed(str))
}

// P forwards to g.gen.P.
func (r *rrpc) P(args ...interface{}) { r.gen.P(args...) }

// Generate generates code for the services in the given file.
func (r *rrpc) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	r.P("// Reference imports to suppress errors if they are not otherwise used.")
	r.P("var _ ", contextPkg, ".Context")
	r.P("var _ ", rrpcPkg, ".Service")
	r.P()
	for i, service := range file.FileDescriptorProto.Service {
		r.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (r *rrpc) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	r.P("import (")
	r.P(contextPkg, " ", strconv.Quote(path.Join(r.gen.ImportPrefix, contextPkgPath)))
	r.P(rrpcPkg, " ", strconv.Quote(path.Join(r.gen.ImportPrefix, rrpcPkgPath)))
	r.P(")")
	r.P()
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in gRPC?
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateClientSignature returns the client-side signature for a method.
func (r *rrpc) generateClientSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	reqArg := "in *" + r.typeName(method.GetInputType()) // In type
	if method.GetClientStreaming() {
		reqArg = "" // what is streaming
	}
	respName := "*" + r.typeName(method.GetOutputType())
	if method.GetServerStreaming() || method.GetClientStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client" // what is streaming
	}
	return fmt.Sprintf("%s(ctx %s.Context, %s) (%s, error)", methName, contextPkg, reqArg, respName)
}

func (r *rrpc) generateClientMethod(servName, fullServName, serviceDescVar string, method *pb.MethodDescriptorProto, descExpr string) {
	qname := fullServName     // Queue name
	dtype := method.GetName() // Delivery type
	//inType := r.typeName(method.GetInputType())
	outType := r.typeName(method.GetOutputType())

	r.P("func (c *", unexport(servName), "Client) ", r.generateClientSignature(servName, method), "{")
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		r.P("out := new(", outType, ")")
		r.P(`err := c.cc.Invoke(ctx, "`, qname, `", "`, dtype, `", in, out)`) // Call service
		r.P("if err != nil { return nil, err }")
		r.P("return out, nil")
		r.P("}")
		r.P()
		return
	}

	// stream ...
	r.P("// FATAL: Streams not supported!")
}

// generateServerSignature returns the server-side signature for a method.
func (r *rrpc) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "error"
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		reqArgs = append(reqArgs, contextPkg+".Context")
		ret = "(*" + r.typeName(method.GetOutputType()) + ", error)"
	}
	if !method.GetClientStreaming() {
		reqArgs = append(reqArgs, "*"+r.typeName(method.GetInputType()))
	}
	if method.GetServerStreaming() || method.GetClientStreaming() {
		reqArgs = append(reqArgs, servName+"_"+generator.CamelCase(origMethName)+"Server")
	}

	return methName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

func (r *rrpc) generateServerMethod(servName string, method *pb.MethodDescriptorProto) string {
	methName := generator.CamelCase(method.GetName())
	hname := fmt.Sprintf("_%s_%s_Handler", servName, methName)
	inType := r.typeName(method.GetInputType())
	//outType := r.typeName(method.GetOutputType())

	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		r.P("func ", hname, "(srv interface{}, ctx ", contextPkg, ".Context, b []byte) ([]byte, error) {")
		r.P("in := new(", inType, ")")
		//r.P("if err := proto.Unmarshal(b, in); err != nil { return nil, err }")
		r.P("if err := ", rrpcPkg, ".Dec(b, in); err != nil { return nil, err }")
		r.P("out, err := srv.(", servName, "Server).", methName, "(ctx, in)")
		r.P("if err != nil { return nil, err }")
		r.P("return ", rrpcPkg, ".Enc(out)")
		//r.P("return b, nil")
		r.P("}")
		r.P()
		return hname
	}

	// stream ...
	r.P("// FATAL: Streams not supported!")
	return hname
}

// generateService generates all the code for the named service.
func (r *rrpc) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	path := fmt.Sprintf("6,%d", index) // 6 means service.

	origServName := service.GetName()
	fullServName := origServName
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "." + fullServName // what is fullServName
	}
	servName := generator.CamelCase(origServName)

	r.P()
	r.P("// Client API for ", servName, " service")
	r.P()

	// Client interface.
	r.P("type ", servName, "Client interface{")
	for i, method := range service.Method {
		r.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service. what is comments
		r.P(r.generateClientSignature(servName, method))
	}
	r.P("}")
	r.P()

	// Client structure.
	r.P("type ", unexport(servName), "Client struct {")
	r.P("cc *", rrpcPkg, ".Service")
	r.P("}")
	r.P()

	// NewClient factory.
	r.P("func New", servName, "Client (cc *", rrpcPkg, ".Service) ", servName, "Client {")
	r.P("return &", unexport(servName), "Client{cc}")
	r.P("}")
	r.P()

	var methodIndex, streamIndex int
	serviceDescVar := "_" + servName + "_serviceDesc"
	// Client method implementaions.
	for _, method := range service.Method {
		var descExpr string

		// TODO: what is this??
		if !method.GetServerStreaming() && !method.GetClientStreaming() {
			// Unary RPC method... what is streaming....
			descExpr = fmt.Sprintf("&%s.Methods[%d]", serviceDescVar, methodIndex)
			methodIndex++
		} else {
			// Streaming RPC method
			descExpr = fmt.Sprintf("&%s.Streams[%d]", serviceDescVar, streamIndex)
			streamIndex++
		}
		r.generateClientMethod(servName, fullServName, serviceDescVar, method, descExpr)
	}

	r.P("// Server API for ", servName, " service")
	r.P()

	// Server interface.
	serverType := servName + "Server"
	r.P("type ", serverType, " interface {")
	for i, method := range service.Method {
		r.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		r.P(r.generateServerSignature(servName, method))
	}
	r.P("}")
	r.P()

	// Server registration.
	r.P("func Register", servName, "Server(s *", rrpcPkg, ".Service, srv ", serverType, ") {")
	r.P("s.RegisterService(&", serviceDescVar, ", srv)")
	r.P("}")
	r.P()

	// Server handler implementations.
	var handlerNames []string
	for _, method := range service.Method {
		hname := r.generateServerMethod(servName, method)
		handlerNames = append(handlerNames, hname)
	}

	// Service descriptor.
	r.P("var ", serviceDescVar, " = ", rrpcPkg, ".ServiceDesc {")
	r.P("ServiceName: ", strconv.Quote(fullServName), ",")
	r.P("HandlerType: (*", serverType, ")(nil),") // TODO: check this
	r.P("Methods: []", rrpcPkg, ".MethodDesc{")
	for i, method := range service.Method {
		if method.GetServerStreaming() || method.GetClientStreaming() {
			continue
		}
		r.P("{")
		r.P("MethodName: ", strconv.Quote(method.GetName()), ",")
		r.P("Handler: ", handlerNames[i], ",")
		r.P("},")
	}
	r.P("},")
	r.P("}")
	r.P()

}
