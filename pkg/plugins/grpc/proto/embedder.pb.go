package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)

	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EmbedRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Text          string                 `protobuf:"bytes,1,opt,name=text,proto3" json:"text,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EmbedRequest) Reset() {
	*x = EmbedRequest{}
	mi := &file_embedder_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EmbedRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmbedRequest) ProtoMessage() {}

func (x *EmbedRequest) ProtoReflect() protoreflect.Message {
	mi := &file_embedder_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*EmbedRequest) Descriptor() ([]byte, []int) {
	return file_embedder_proto_rawDescGZIP(), []int{0}
}

func (x *EmbedRequest) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

type EmbedResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Vector        []float32              `protobuf:"fixed32,1,rep,packed,name=vector,proto3" json:"vector,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EmbedResponse) Reset() {
	*x = EmbedResponse{}
	mi := &file_embedder_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EmbedResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmbedResponse) ProtoMessage() {}

func (x *EmbedResponse) ProtoReflect() protoreflect.Message {
	mi := &file_embedder_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*EmbedResponse) Descriptor() ([]byte, []int) {
	return file_embedder_proto_rawDescGZIP(), []int{1}
}

func (x *EmbedResponse) GetVector() []float32 {
	if x != nil {
		return x.Vector
	}
	return nil
}

func (x *EmbedResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type EmbedderInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ModelName     string                 `protobuf:"bytes,1,opt,name=model_name,json=modelName,proto3" json:"model_name,omitempty"`
	Dimension     int32                  `protobuf:"varint,2,opt,name=dimension,proto3" json:"dimension,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EmbedderInfo) Reset() {
	*x = EmbedderInfo{}
	mi := &file_embedder_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EmbedderInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmbedderInfo) ProtoMessage() {}

func (x *EmbedderInfo) ProtoReflect() protoreflect.Message {
	mi := &file_embedder_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*EmbedderInfo) Descriptor() ([]byte, []int) {
	return file_embedder_proto_rawDescGZIP(), []int{2}
}

func (x *EmbedderInfo) GetModelName() string {
	if x != nil {
		return x.ModelName
	}
	return ""
}

func (x *EmbedderInfo) GetDimension() int32 {
	if x != nil {
		return x.Dimension
	}
	return 0
}

var File_embedder_proto protoreflect.FileDescriptor

const file_embedder_proto_rawDesc = "" +
	"\n" +
	"\x0eembedder.proto\x12\rhector.plugin\x1a\fcommon.proto\"\"\n" +
	"\fEmbedRequest\x12\x12\n" +
	"\x04text\x18\x01 \x01(\tR\x04text\"=\n" +
	"\rEmbedResponse\x12\x16\n" +
	"\x06vector\x18\x01 \x03(\x02R\x06vector\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"K\n" +
	"\fEmbedderInfo\x12\x1d\n" +
	"\n" +
	"model_name\x18\x01 \x01(\tR\tmodelName\x12\x1c\n" +
	"\tdimension\x18\x02 \x01(\x05R\tdimension2\x9d\x04\n" +
	"\x10EmbedderProvider\x12Q\n" +
	"\n" +
	"Initialize\x12 .hector.plugin.InitializeRequest\x1a!.hector.plugin.InitializeResponse\x12K\n" +
	"\bShutdown\x12\x1e.hector.plugin.ShutdownRequest\x1a\x1f.hector.plugin.ShutdownResponse\x12E\n" +
	"\x06Health\x12\x1c.hector.plugin.HealthRequest\x1a\x1d.hector.plugin.HealthResponse\x12N\n" +
	"\vGetManifest\x12\x1e.hector.plugin.ManifestRequest\x1a\x1f.hector.plugin.ManifestResponse\x12H\n" +
	"\tGetStatus\x12\x1c.hector.plugin.StatusRequest\x1a\x1d.hector.plugin.StatusResponse\x12B\n" +
	"\x05Embed\x12\x1b.hector.plugin.EmbedRequest\x1a\x1c.hector.plugin.EmbedResponse\x12D\n" +
	"\x0fGetEmbedderInfo\x12\x14.hector.plugin.Empty\x1a\x1b.hector.plugin.EmbedderInfoB1Z/github.com/kadirpekel/hector/plugins/grpc/protob\x06proto3"

var (
	file_embedder_proto_rawDescOnce sync.Once
	file_embedder_proto_rawDescData []byte
)

func file_embedder_proto_rawDescGZIP() []byte {
	file_embedder_proto_rawDescOnce.Do(func() {
		file_embedder_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_embedder_proto_rawDesc), len(file_embedder_proto_rawDesc)))
	})
	return file_embedder_proto_rawDescData
}

var file_embedder_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_embedder_proto_goTypes = []any{
	(*EmbedRequest)(nil),
	(*EmbedResponse)(nil),
	(*EmbedderInfo)(nil),
	(*InitializeRequest)(nil),
	(*ShutdownRequest)(nil),
	(*HealthRequest)(nil),
	(*ManifestRequest)(nil),
	(*StatusRequest)(nil),
	(*Empty)(nil),
	(*InitializeResponse)(nil),
	(*ShutdownResponse)(nil),
	(*HealthResponse)(nil),
	(*ManifestResponse)(nil),
	(*StatusResponse)(nil),
}
var file_embedder_proto_depIdxs = []int32{
	3,
	4,
	5,
	6,
	7,
	0,
	8,
	9,
	10,
	11,
	12,
	13,
	1,
	2,
	7,
	0,
	0,
	0,
	0,
}

func init() { file_embedder_proto_init() }
func file_embedder_proto_init() {
	if File_embedder_proto != nil {
		return
	}
	file_common_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_embedder_proto_rawDesc), len(file_embedder_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_embedder_proto_goTypes,
		DependencyIndexes: file_embedder_proto_depIdxs,
		MessageInfos:      file_embedder_proto_msgTypes,
	}.Build()
	File_embedder_proto = out.File
	file_embedder_proto_goTypes = nil
	file_embedder_proto_depIdxs = nil
}
