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

type Empty struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Empty) Reset() {
	*x = Empty{}
	mi := &file_common_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Empty) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Empty) ProtoMessage() {}

func (x *Empty) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*Empty) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{0}
}

type InitializeRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Config        map[string]string      `protobuf:"bytes,1,rep,name=config,proto3" json:"config,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *InitializeRequest) Reset() {
	*x = InitializeRequest{}
	mi := &file_common_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *InitializeRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitializeRequest) ProtoMessage() {}

func (x *InitializeRequest) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*InitializeRequest) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{1}
}

func (x *InitializeRequest) GetConfig() map[string]string {
	if x != nil {
		return x.Config
	}
	return nil
}

type InitializeResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *InitializeResponse) Reset() {
	*x = InitializeResponse{}
	mi := &file_common_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *InitializeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitializeResponse) ProtoMessage() {}

func (x *InitializeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*InitializeResponse) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{2}
}

func (x *InitializeResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *InitializeResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type ShutdownRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ShutdownRequest) Reset() {
	*x = ShutdownRequest{}
	mi := &file_common_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ShutdownRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownRequest) ProtoMessage() {}

func (x *ShutdownRequest) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*ShutdownRequest) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{3}
}

type ShutdownResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ShutdownResponse) Reset() {
	*x = ShutdownResponse{}
	mi := &file_common_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ShutdownResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ShutdownResponse) ProtoMessage() {}

func (x *ShutdownResponse) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*ShutdownResponse) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{4}
}

func (x *ShutdownResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *ShutdownResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type HealthRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HealthRequest) Reset() {
	*x = HealthRequest{}
	mi := &file_common_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HealthRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HealthRequest) ProtoMessage() {}

func (x *HealthRequest) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*HealthRequest) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{5}
}

type HealthResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Healthy       bool                   `protobuf:"varint,1,opt,name=healthy,proto3" json:"healthy,omitempty"`
	Message       string                 `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HealthResponse) Reset() {
	*x = HealthResponse{}
	mi := &file_common_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HealthResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HealthResponse) ProtoMessage() {}

func (x *HealthResponse) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*HealthResponse) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{6}
}

func (x *HealthResponse) GetHealthy() bool {
	if x != nil {
		return x.Healthy
	}
	return false
}

func (x *HealthResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type ManifestRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ManifestRequest) Reset() {
	*x = ManifestRequest{}
	mi := &file_common_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ManifestRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ManifestRequest) ProtoMessage() {}

func (x *ManifestRequest) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*ManifestRequest) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{7}
}

type ManifestResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Version       string                 `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	Author        string                 `protobuf:"bytes,3,opt,name=author,proto3" json:"author,omitempty"`
	Description   string                 `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	Type          string                 `protobuf:"bytes,5,opt,name=type,proto3" json:"type,omitempty"`
	Protocol      string                 `protobuf:"bytes,6,opt,name=protocol,proto3" json:"protocol,omitempty"`
	HectorVersion string                 `protobuf:"bytes,7,opt,name=hector_version,json=hectorVersion,proto3" json:"hector_version,omitempty"`
	Capabilities  map[string]string      `protobuf:"bytes,8,rep,name=capabilities,proto3" json:"capabilities,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ManifestResponse) Reset() {
	*x = ManifestResponse{}
	mi := &file_common_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ManifestResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ManifestResponse) ProtoMessage() {}

func (x *ManifestResponse) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*ManifestResponse) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{8}
}

func (x *ManifestResponse) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ManifestResponse) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

func (x *ManifestResponse) GetAuthor() string {
	if x != nil {
		return x.Author
	}
	return ""
}

func (x *ManifestResponse) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *ManifestResponse) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *ManifestResponse) GetProtocol() string {
	if x != nil {
		return x.Protocol
	}
	return ""
}

func (x *ManifestResponse) GetHectorVersion() string {
	if x != nil {
		return x.HectorVersion
	}
	return ""
}

func (x *ManifestResponse) GetCapabilities() map[string]string {
	if x != nil {
		return x.Capabilities
	}
	return nil
}

type StatusRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StatusRequest) Reset() {
	*x = StatusRequest{}
	mi := &file_common_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusRequest) ProtoMessage() {}

func (x *StatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*StatusRequest) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{9}
}

type StatusResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Status        string                 `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StatusResponse) Reset() {
	*x = StatusResponse{}
	mi := &file_common_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StatusResponse) ProtoMessage() {}

func (x *StatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_common_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*StatusResponse) Descriptor() ([]byte, []int) {
	return file_common_proto_rawDescGZIP(), []int{10}
}

func (x *StatusResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

var File_common_proto protoreflect.FileDescriptor

const file_common_proto_rawDesc = "" +
	"\n" +
	"\fcommon.proto\x12\rhector.plugin\"\a\n" +
	"\x05Empty\"\x94\x01\n" +
	"\x11InitializeRequest\x12D\n" +
	"\x06config\x18\x01 \x03(\v2,.hector.plugin.InitializeRequest.ConfigEntryR\x06config\x1a9\n" +
	"\vConfigEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"D\n" +
	"\x12InitializeResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"\x11\n" +
	"\x0fShutdownRequest\"B\n" +
	"\x10ShutdownResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"\x0f\n" +
	"\rHealthRequest\"D\n" +
	"\x0eHealthResponse\x12\x18\n" +
	"\ahealthy\x18\x01 \x01(\bR\ahealthy\x12\x18\n" +
	"\amessage\x18\x02 \x01(\tR\amessage\"\x11\n" +
	"\x0fManifestRequest\"\xe9\x02\n" +
	"\x10ManifestResponse\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x18\n" +
	"\aversion\x18\x02 \x01(\tR\aversion\x12\x16\n" +
	"\x06author\x18\x03 \x01(\tR\x06author\x12 \n" +
	"\vdescription\x18\x04 \x01(\tR\vdescription\x12\x12\n" +
	"\x04type\x18\x05 \x01(\tR\x04type\x12\x1a\n" +
	"\bprotocol\x18\x06 \x01(\tR\bprotocol\x12%\n" +
	"\x0ehector_version\x18\a \x01(\tR\rhectorVersion\x12U\n" +
	"\fcapabilities\x18\b \x03(\v21.hector.plugin.ManifestResponse.CapabilitiesEntryR\fcapabilities\x1a?\n" +
	"\x11CapabilitiesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"\x0f\n" +
	"\rStatusRequest\"(\n" +
	"\x0eStatusResponse\x12\x16\n" +
	"\x06status\x18\x01 \x01(\tR\x06statusB1Z/github.com/kadirpekel/hector/plugins/grpc/protob\x06proto3"

var (
	file_common_proto_rawDescOnce sync.Once
	file_common_proto_rawDescData []byte
)

func file_common_proto_rawDescGZIP() []byte {
	file_common_proto_rawDescOnce.Do(func() {
		file_common_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_common_proto_rawDesc), len(file_common_proto_rawDesc)))
	})
	return file_common_proto_rawDescData
}

var file_common_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_common_proto_goTypes = []any{
	(*Empty)(nil),
	(*InitializeRequest)(nil),
	(*InitializeResponse)(nil),
	(*ShutdownRequest)(nil),
	(*ShutdownResponse)(nil),
	(*HealthRequest)(nil),
	(*HealthResponse)(nil),
	(*ManifestRequest)(nil),
	(*ManifestResponse)(nil),
	(*StatusRequest)(nil),
	(*StatusResponse)(nil),
	nil,
	nil,
}
var file_common_proto_depIdxs = []int32{
	11,
	12,
	2,
	2,
	2,
	2,
	0,
}

func init() { file_common_proto_init() }
func file_common_proto_init() {
	if File_common_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_common_proto_rawDesc), len(file_common_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_common_proto_goTypes,
		DependencyIndexes: file_common_proto_depIdxs,
		MessageInfos:      file_common_proto_msgTypes,
	}.Build()
	File_common_proto = out.File
	file_common_proto_goTypes = nil
	file_common_proto_depIdxs = nil
}
