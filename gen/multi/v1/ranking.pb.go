// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: multi/v1/ranking.proto

package multiv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetRankingRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId        int64  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	CharacterName string `protobuf:"bytes,2,opt,name=character_name,json=characterName,proto3" json:"character_name,omitempty"`
	ClassType     int64  `protobuf:"varint,3,opt,name=class_type,json=classType,proto3" json:"class_type,omitempty"`
	Offset        int64  `protobuf:"varint,4,opt,name=offset,proto3" json:"offset,omitempty"`
}

func (x *GetRankingRequest) Reset() {
	*x = GetRankingRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_ranking_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetRankingRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRankingRequest) ProtoMessage() {}

func (x *GetRankingRequest) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_ranking_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRankingRequest.ProtoReflect.Descriptor instead.
func (*GetRankingRequest) Descriptor() ([]byte, []int) {
	return file_multi_v1_ranking_proto_rawDescGZIP(), []int{0}
}

func (x *GetRankingRequest) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *GetRankingRequest) GetCharacterName() string {
	if x != nil {
		return x.CharacterName
	}
	return ""
}

func (x *GetRankingRequest) GetClassType() int64 {
	if x != nil {
		return x.ClassType
	}
	return 0
}

func (x *GetRankingRequest) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

type GetRankingResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CurrentPlayer *RankingPosition   `protobuf:"bytes,1,opt,name=CurrentPlayer,proto3" json:"CurrentPlayer,omitempty"`
	Players       []*RankingPosition `protobuf:"bytes,2,rep,name=Players,proto3" json:"Players,omitempty"`
}

func (x *GetRankingResponse) Reset() {
	*x = GetRankingResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_ranking_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetRankingResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRankingResponse) ProtoMessage() {}

func (x *GetRankingResponse) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_ranking_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetRankingResponse.ProtoReflect.Descriptor instead.
func (*GetRankingResponse) Descriptor() ([]byte, []int) {
	return file_multi_v1_ranking_proto_rawDescGZIP(), []int{1}
}

func (x *GetRankingResponse) GetCurrentPlayer() *RankingPosition {
	if x != nil {
		return x.CurrentPlayer
	}
	return nil
}

func (x *GetRankingResponse) GetPlayers() []*RankingPosition {
	if x != nil {
		return x.Players
	}
	return nil
}

type RankingPosition struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Rank          uint32 `protobuf:"varint,1,opt,name=rank,proto3" json:"rank,omitempty"`
	Points        uint32 `protobuf:"varint,2,opt,name=points,proto3" json:"points,omitempty"`
	Username      string `protobuf:"bytes,3,opt,name=username,proto3" json:"username,omitempty"`
	CharacterName string `protobuf:"bytes,4,opt,name=character_name,json=characterName,proto3" json:"character_name,omitempty"`
}

func (x *RankingPosition) Reset() {
	*x = RankingPosition{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_ranking_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RankingPosition) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RankingPosition) ProtoMessage() {}

func (x *RankingPosition) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_ranking_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RankingPosition.ProtoReflect.Descriptor instead.
func (*RankingPosition) Descriptor() ([]byte, []int) {
	return file_multi_v1_ranking_proto_rawDescGZIP(), []int{2}
}

func (x *RankingPosition) GetRank() uint32 {
	if x != nil {
		return x.Rank
	}
	return 0
}

func (x *RankingPosition) GetPoints() uint32 {
	if x != nil {
		return x.Points
	}
	return 0
}

func (x *RankingPosition) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *RankingPosition) GetCharacterName() string {
	if x != nil {
		return x.CharacterName
	}
	return ""
}

var File_multi_v1_ranking_proto protoreflect.FileDescriptor

var file_multi_v1_ranking_proto_rawDesc = []byte{
	0x0a, 0x16, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x61, 0x6e, 0x6b, 0x69,
	0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e,
	0x76, 0x31, 0x22, 0x8a, 0x01, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x52, 0x61, 0x6e, 0x6b, 0x69, 0x6e,
	0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49,
	0x64, 0x12, 0x25, 0x0a, 0x0e, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x63, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x6c, 0x61, 0x73,
	0x73, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x63, 0x6c,
	0x61, 0x73, 0x73, 0x54, 0x79, 0x70, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65,
	0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x22,
	0x8a, 0x01, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x52, 0x61, 0x6e, 0x6b, 0x69, 0x6e, 0x67, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3f, 0x0a, 0x0d, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e,
	0x74, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e,
	0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x61, 0x6e, 0x6b, 0x69, 0x6e, 0x67,
	0x50, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0d, 0x43, 0x75, 0x72, 0x72, 0x65, 0x6e,
	0x74, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x12, 0x33, 0x0a, 0x07, 0x50, 0x6c, 0x61, 0x79, 0x65,
	0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69,
	0x2e, 0x76, 0x31, 0x2e, 0x52, 0x61, 0x6e, 0x6b, 0x69, 0x6e, 0x67, 0x50, 0x6f, 0x73, 0x69, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x07, 0x50, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x73, 0x22, 0x80, 0x01, 0x0a,
	0x0f, 0x52, 0x61, 0x6e, 0x6b, 0x69, 0x6e, 0x67, 0x50, 0x6f, 0x73, 0x69, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x12, 0x0a, 0x04, 0x72, 0x61, 0x6e, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04,
	0x72, 0x61, 0x6e, 0x6b, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x06, 0x70, 0x6f, 0x69, 0x6e, 0x74, 0x73, 0x12, 0x1a, 0x0a, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08,
	0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x25, 0x0a, 0x0e, 0x63, 0x68, 0x61, 0x72,
	0x61, 0x63, 0x74, 0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0d, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x32,
	0x5b, 0x0a, 0x0e, 0x52, 0x61, 0x6e, 0x6b, 0x69, 0x6e, 0x67, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x12, 0x49, 0x0a, 0x0a, 0x47, 0x65, 0x74, 0x52, 0x61, 0x6e, 0x6b, 0x69, 0x6e, 0x67, 0x12,
	0x1b, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x61,
	0x6e, 0x6b, 0x69, 0x6e, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x6d,
	0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x52, 0x61, 0x6e, 0x6b, 0x69,
	0x6e, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x95, 0x01, 0x0a,
	0x0c, 0x63, 0x6f, 0x6d, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x42, 0x0c, 0x52,
	0x61, 0x6e, 0x6b, 0x69, 0x6e, 0x67, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x36, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x69, 0x73, 0x70, 0x65, 0x6c,
	0x2d, 0x72, 0x65, 0x2f, 0x64, 0x69, 0x73, 0x70, 0x65, 0x6c, 0x2d, 0x6d, 0x75, 0x6c, 0x74, 0x69,
	0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2f, 0x76, 0x31, 0x3b, 0x6d, 0x75,
	0x6c, 0x74, 0x69, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x4d, 0x58, 0x58, 0xaa, 0x02, 0x08, 0x4d, 0x75,
	0x6c, 0x74, 0x69, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x08, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x5c, 0x56,
	0x31, 0xe2, 0x02, 0x14, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42,
	0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x09, 0x4d, 0x75, 0x6c, 0x74, 0x69,
	0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_multi_v1_ranking_proto_rawDescOnce sync.Once
	file_multi_v1_ranking_proto_rawDescData = file_multi_v1_ranking_proto_rawDesc
)

func file_multi_v1_ranking_proto_rawDescGZIP() []byte {
	file_multi_v1_ranking_proto_rawDescOnce.Do(func() {
		file_multi_v1_ranking_proto_rawDescData = protoimpl.X.CompressGZIP(file_multi_v1_ranking_proto_rawDescData)
	})
	return file_multi_v1_ranking_proto_rawDescData
}

var file_multi_v1_ranking_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_multi_v1_ranking_proto_goTypes = []interface{}{
	(*GetRankingRequest)(nil),  // 0: multi.v1.GetRankingRequest
	(*GetRankingResponse)(nil), // 1: multi.v1.GetRankingResponse
	(*RankingPosition)(nil),    // 2: multi.v1.RankingPosition
}
var file_multi_v1_ranking_proto_depIdxs = []int32{
	2, // 0: multi.v1.GetRankingResponse.CurrentPlayer:type_name -> multi.v1.RankingPosition
	2, // 1: multi.v1.GetRankingResponse.Players:type_name -> multi.v1.RankingPosition
	0, // 2: multi.v1.RankingService.GetRanking:input_type -> multi.v1.GetRankingRequest
	1, // 3: multi.v1.RankingService.GetRanking:output_type -> multi.v1.GetRankingResponse
	3, // [3:4] is the sub-list for method output_type
	2, // [2:3] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_multi_v1_ranking_proto_init() }
func file_multi_v1_ranking_proto_init() {
	if File_multi_v1_ranking_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_multi_v1_ranking_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetRankingRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_multi_v1_ranking_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetRankingResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_multi_v1_ranking_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RankingPosition); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_multi_v1_ranking_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_multi_v1_ranking_proto_goTypes,
		DependencyIndexes: file_multi_v1_ranking_proto_depIdxs,
		MessageInfos:      file_multi_v1_ranking_proto_msgTypes,
	}.Build()
	File_multi_v1_ranking_proto = out.File
	file_multi_v1_ranking_proto_rawDesc = nil
	file_multi_v1_ranking_proto_goTypes = nil
	file_multi_v1_ranking_proto_depIdxs = nil
}
