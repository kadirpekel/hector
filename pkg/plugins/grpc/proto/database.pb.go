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

type UpsertRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Collection    string                 `protobuf:"bytes,1,opt,name=collection,proto3" json:"collection,omitempty"`
	Id            string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Vector        []float32              `protobuf:"fixed32,3,rep,packed,name=vector,proto3" json:"vector,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,4,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpsertRequest) Reset() {
	*x = UpsertRequest{}
	mi := &file_database_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpsertRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpsertRequest) ProtoMessage() {}

func (x *UpsertRequest) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*UpsertRequest) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{0}
}

func (x *UpsertRequest) GetCollection() string {
	if x != nil {
		return x.Collection
	}
	return ""
}

func (x *UpsertRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UpsertRequest) GetVector() []float32 {
	if x != nil {
		return x.Vector
	}
	return nil
}

func (x *UpsertRequest) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type UpsertResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpsertResponse) Reset() {
	*x = UpsertResponse{}
	mi := &file_database_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpsertResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpsertResponse) ProtoMessage() {}

func (x *UpsertResponse) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*UpsertResponse) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{1}
}

func (x *UpsertResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *UpsertResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type SearchRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Collection    string                 `protobuf:"bytes,1,opt,name=collection,proto3" json:"collection,omitempty"`
	Vector        []float32              `protobuf:"fixed32,2,rep,packed,name=vector,proto3" json:"vector,omitempty"`
	TopK          int32                  `protobuf:"varint,3,opt,name=top_k,json=topK,proto3" json:"top_k,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SearchRequest) Reset() {
	*x = SearchRequest{}
	mi := &file_database_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SearchRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SearchRequest) ProtoMessage() {}

func (x *SearchRequest) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*SearchRequest) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{2}
}

func (x *SearchRequest) GetCollection() string {
	if x != nil {
		return x.Collection
	}
	return ""
}

func (x *SearchRequest) GetVector() []float32 {
	if x != nil {
		return x.Vector
	}
	return nil
}

func (x *SearchRequest) GetTopK() int32 {
	if x != nil {
		return x.TopK
	}
	return 0
}

type SearchResult struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Score         float32                `protobuf:"fixed32,2,opt,name=score,proto3" json:"score,omitempty"`
	Content       string                 `protobuf:"bytes,3,opt,name=content,proto3" json:"content,omitempty"`
	Vector        []float32              `protobuf:"fixed32,4,rep,packed,name=vector,proto3" json:"vector,omitempty"`
	Metadata      map[string]string      `protobuf:"bytes,5,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	ModelName     string                 `protobuf:"bytes,6,opt,name=model_name,json=modelName,proto3" json:"model_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SearchResult) Reset() {
	*x = SearchResult{}
	mi := &file_database_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SearchResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SearchResult) ProtoMessage() {}

func (x *SearchResult) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*SearchResult) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{3}
}

func (x *SearchResult) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *SearchResult) GetScore() float32 {
	if x != nil {
		return x.Score
	}
	return 0
}

func (x *SearchResult) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *SearchResult) GetVector() []float32 {
	if x != nil {
		return x.Vector
	}
	return nil
}

func (x *SearchResult) GetMetadata() map[string]string {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *SearchResult) GetModelName() string {
	if x != nil {
		return x.ModelName
	}
	return ""
}

type SearchResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Results       []*SearchResult        `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SearchResponse) Reset() {
	*x = SearchResponse{}
	mi := &file_database_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SearchResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SearchResponse) ProtoMessage() {}

func (x *SearchResponse) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*SearchResponse) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{4}
}

func (x *SearchResponse) GetResults() []*SearchResult {
	if x != nil {
		return x.Results
	}
	return nil
}

func (x *SearchResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type DeleteRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Collection    string                 `protobuf:"bytes,1,opt,name=collection,proto3" json:"collection,omitempty"`
	Id            string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteRequest) Reset() {
	*x = DeleteRequest{}
	mi := &file_database_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRequest) ProtoMessage() {}

func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{5}
}

func (x *DeleteRequest) GetCollection() string {
	if x != nil {
		return x.Collection
	}
	return ""
}

func (x *DeleteRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type DeleteResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteResponse) Reset() {
	*x = DeleteResponse{}
	mi := &file_database_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteResponse) ProtoMessage() {}

func (x *DeleteResponse) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*DeleteResponse) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{6}
}

func (x *DeleteResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *DeleteResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type CreateCollectionRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Collection    string                 `protobuf:"bytes,1,opt,name=collection,proto3" json:"collection,omitempty"`
	VectorSize    uint64                 `protobuf:"varint,2,opt,name=vector_size,json=vectorSize,proto3" json:"vector_size,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateCollectionRequest) Reset() {
	*x = CreateCollectionRequest{}
	mi := &file_database_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateCollectionRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateCollectionRequest) ProtoMessage() {}

func (x *CreateCollectionRequest) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*CreateCollectionRequest) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{7}
}

func (x *CreateCollectionRequest) GetCollection() string {
	if x != nil {
		return x.Collection
	}
	return ""
}

func (x *CreateCollectionRequest) GetVectorSize() uint64 {
	if x != nil {
		return x.VectorSize
	}
	return 0
}

type CreateCollectionResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateCollectionResponse) Reset() {
	*x = CreateCollectionResponse{}
	mi := &file_database_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateCollectionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateCollectionResponse) ProtoMessage() {}

func (x *CreateCollectionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*CreateCollectionResponse) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{8}
}

func (x *CreateCollectionResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *CreateCollectionResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type DeleteCollectionRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Collection    string                 `protobuf:"bytes,1,opt,name=collection,proto3" json:"collection,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteCollectionRequest) Reset() {
	*x = DeleteCollectionRequest{}
	mi := &file_database_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteCollectionRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteCollectionRequest) ProtoMessage() {}

func (x *DeleteCollectionRequest) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*DeleteCollectionRequest) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{9}
}

func (x *DeleteCollectionRequest) GetCollection() string {
	if x != nil {
		return x.Collection
	}
	return ""
}

type DeleteCollectionResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteCollectionResponse) Reset() {
	*x = DeleteCollectionResponse{}
	mi := &file_database_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteCollectionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteCollectionResponse) ProtoMessage() {}

func (x *DeleteCollectionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_database_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*DeleteCollectionResponse) Descriptor() ([]byte, []int) {
	return file_database_proto_rawDescGZIP(), []int{10}
}

func (x *DeleteCollectionResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *DeleteCollectionResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

var File_database_proto protoreflect.FileDescriptor

const file_database_proto_rawDesc = "" +
	"\n" +
	"\x0edatabase.proto\x12\rhector.plugin\x1a\fcommon.proto\"\xdc\x01\n" +
	"\rUpsertRequest\x12\x1e\n" +
	"\n" +
	"collection\x18\x01 \x01(\tR\n" +
	"collection\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\x12\x16\n" +
	"\x06vector\x18\x03 \x03(\x02R\x06vector\x12F\n" +
	"\bmetadata\x18\x04 \x03(\v2*.hector.plugin.UpsertRequest.MetadataEntryR\bmetadata\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"@\n" +
	"\x0eUpsertResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"\\\n" +
	"\rSearchRequest\x12\x1e\n" +
	"\n" +
	"collection\x18\x01 \x01(\tR\n" +
	"collection\x12\x16\n" +
	"\x06vector\x18\x02 \x03(\x02R\x06vector\x12\x13\n" +
	"\x05top_k\x18\x03 \x01(\x05R\x04topK\"\x89\x02\n" +
	"\fSearchResult\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x14\n" +
	"\x05score\x18\x02 \x01(\x02R\x05score\x12\x18\n" +
	"\acontent\x18\x03 \x01(\tR\acontent\x12\x16\n" +
	"\x06vector\x18\x04 \x03(\x02R\x06vector\x12E\n" +
	"\bmetadata\x18\x05 \x03(\v2).hector.plugin.SearchResult.MetadataEntryR\bmetadata\x12\x1d\n" +
	"\n" +
	"model_name\x18\x06 \x01(\tR\tmodelName\x1a;\n" +
	"\rMetadataEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"]\n" +
	"\x0eSearchResponse\x125\n" +
	"\aresults\x18\x01 \x03(\v2\x1b.hector.plugin.SearchResultR\aresults\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"?\n" +
	"\rDeleteRequest\x12\x1e\n" +
	"\n" +
	"collection\x18\x01 \x01(\tR\n" +
	"collection\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\"@\n" +
	"\x0eDeleteResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"Z\n" +
	"\x17CreateCollectionRequest\x12\x1e\n" +
	"\n" +
	"collection\x18\x01 \x01(\tR\n" +
	"collection\x12\x1f\n" +
	"\vvector_size\x18\x02 \x01(\x04R\n" +
	"vectorSize\"J\n" +
	"\x18CreateCollectionResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error\"9\n" +
	"\x17DeleteCollectionRequest\x12\x1e\n" +
	"\n" +
	"collection\x18\x01 \x01(\tR\n" +
	"collection\"J\n" +
	"\x18DeleteCollectionResponse\x12\x18\n" +
	"\asuccess\x18\x01 \x01(\bR\asuccess\x12\x14\n" +
	"\x05error\x18\x02 \x01(\tR\x05error2\xb2\x06\n" +
	"\x10DatabaseProvider\x12Q\n" +
	"\n" +
	"Initialize\x12 .hector.plugin.InitializeRequest\x1a!.hector.plugin.InitializeResponse\x12K\n" +
	"\bShutdown\x12\x1e.hector.plugin.ShutdownRequest\x1a\x1f.hector.plugin.ShutdownResponse\x12E\n" +
	"\x06Health\x12\x1c.hector.plugin.HealthRequest\x1a\x1d.hector.plugin.HealthResponse\x12N\n" +
	"\vGetManifest\x12\x1e.hector.plugin.ManifestRequest\x1a\x1f.hector.plugin.ManifestResponse\x12H\n" +
	"\tGetStatus\x12\x1c.hector.plugin.StatusRequest\x1a\x1d.hector.plugin.StatusResponse\x12E\n" +
	"\x06Upsert\x12\x1c.hector.plugin.UpsertRequest\x1a\x1d.hector.plugin.UpsertResponse\x12E\n" +
	"\x06Search\x12\x1c.hector.plugin.SearchRequest\x1a\x1d.hector.plugin.SearchResponse\x12E\n" +
	"\x06Delete\x12\x1c.hector.plugin.DeleteRequest\x1a\x1d.hector.plugin.DeleteResponse\x12c\n" +
	"\x10CreateCollection\x12&.hector.plugin.CreateCollectionRequest\x1a'.hector.plugin.CreateCollectionResponse\x12c\n" +
	"\x10DeleteCollection\x12&.hector.plugin.DeleteCollectionRequest\x1a'.hector.plugin.DeleteCollectionResponseB1Z/github.com/kadirpekel/hector/plugins/grpc/protob\x06proto3"

var (
	file_database_proto_rawDescOnce sync.Once
	file_database_proto_rawDescData []byte
)

func file_database_proto_rawDescGZIP() []byte {
	file_database_proto_rawDescOnce.Do(func() {
		file_database_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_database_proto_rawDesc), len(file_database_proto_rawDesc)))
	})
	return file_database_proto_rawDescData
}

var file_database_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_database_proto_goTypes = []any{
	(*UpsertRequest)(nil),
	(*UpsertResponse)(nil),
	(*SearchRequest)(nil),
	(*SearchResult)(nil),
	(*SearchResponse)(nil),
	(*DeleteRequest)(nil),
	(*DeleteResponse)(nil),
	(*CreateCollectionRequest)(nil),
	(*CreateCollectionResponse)(nil),
	(*DeleteCollectionRequest)(nil),
	(*DeleteCollectionResponse)(nil),
	nil,
	nil,
	(*InitializeRequest)(nil),
	(*ShutdownRequest)(nil),
	(*HealthRequest)(nil),
	(*ManifestRequest)(nil),
	(*StatusRequest)(nil),
	(*InitializeResponse)(nil),
	(*ShutdownResponse)(nil),
	(*HealthResponse)(nil),
	(*ManifestResponse)(nil),
	(*StatusResponse)(nil),
}
var file_database_proto_depIdxs = []int32{
	11,
	12,
	3,
	13,
	14,
	15,
	16,
	17,
	0,
	2,
	5,
	7,
	9,
	18,
	19,
	20,
	21,
	22,
	1,
	4,
	6,
	8,
	10,
	13,
	3,
	3,
	3,
	0,
}

func init() { file_database_proto_init() }
func file_database_proto_init() {
	if File_database_proto != nil {
		return
	}
	file_common_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_database_proto_rawDesc), len(file_database_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_database_proto_goTypes,
		DependencyIndexes: file_database_proto_depIdxs,
		MessageInfos:      file_database_proto_msgTypes,
	}.Build()
	File_database_proto = out.File
	file_database_proto_goTypes = nil
	file_database_proto_depIdxs = nil
}
