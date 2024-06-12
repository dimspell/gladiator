// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: multi/v1/channel.proto

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

type Channels int32

const (
	Channels_DISPEL Channels = 0
)

// Enum value maps for Channels.
var (
	Channels_name = map[int32]string{
		0: "DISPEL",
	}
	Channels_value = map[string]int32{
		"DISPEL": 0,
	}
)

func (x Channels) Enum() *Channels {
	p := new(Channels)
	*p = x
	return p
}

func (x Channels) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Channels) Descriptor() protoreflect.EnumDescriptor {
	return file_multi_v1_channel_proto_enumTypes[0].Descriptor()
}

func (Channels) Type() protoreflect.EnumType {
	return &file_multi_v1_channel_proto_enumTypes[0]
}

func (x Channels) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Channels.Descriptor instead.
func (Channels) EnumDescriptor() ([]byte, []int) {
	return file_multi_v1_channel_proto_rawDescGZIP(), []int{0}
}

var File_multi_v1_channel_proto protoreflect.FileDescriptor

var file_multi_v1_channel_proto_rawDesc = []byte{
	0x0a, 0x16, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e,
	0x76, 0x31, 0x2a, 0x16, 0x0a, 0x08, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x73, 0x12, 0x0a,
	0x0a, 0x06, 0x44, 0x49, 0x53, 0x50, 0x45, 0x4c, 0x10, 0x00, 0x42, 0x95, 0x01, 0x0a, 0x0c, 0x63,
	0x6f, 0x6d, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x42, 0x0c, 0x43, 0x68, 0x61,
	0x6e, 0x6e, 0x65, 0x6c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x36, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x69, 0x73, 0x70, 0x65, 0x6c, 0x2d, 0x72,
	0x65, 0x2f, 0x64, 0x69, 0x73, 0x70, 0x65, 0x6c, 0x2d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2f, 0x67,
	0x65, 0x6e, 0x2f, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2f, 0x76, 0x31, 0x3b, 0x6d, 0x75, 0x6c, 0x74,
	0x69, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x4d, 0x58, 0x58, 0xaa, 0x02, 0x08, 0x4d, 0x75, 0x6c, 0x74,
	0x69, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x08, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x5c, 0x56, 0x31, 0xe2,
	0x02, 0x14, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x09, 0x4d, 0x75, 0x6c, 0x74, 0x69, 0x3a, 0x3a,
	0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_multi_v1_channel_proto_rawDescOnce sync.Once
	file_multi_v1_channel_proto_rawDescData = file_multi_v1_channel_proto_rawDesc
)

func file_multi_v1_channel_proto_rawDescGZIP() []byte {
	file_multi_v1_channel_proto_rawDescOnce.Do(func() {
		file_multi_v1_channel_proto_rawDescData = protoimpl.X.CompressGZIP(file_multi_v1_channel_proto_rawDescData)
	})
	return file_multi_v1_channel_proto_rawDescData
}

var file_multi_v1_channel_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_multi_v1_channel_proto_goTypes = []any{
	(Channels)(0), // 0: multi.v1.Channels
}
var file_multi_v1_channel_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_multi_v1_channel_proto_init() }
func file_multi_v1_channel_proto_init() {
	if File_multi_v1_channel_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_multi_v1_channel_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_multi_v1_channel_proto_goTypes,
		DependencyIndexes: file_multi_v1_channel_proto_depIdxs,
		EnumInfos:         file_multi_v1_channel_proto_enumTypes,
	}.Build()
	File_multi_v1_channel_proto = out.File
	file_multi_v1_channel_proto_rawDesc = nil
	file_multi_v1_channel_proto_goTypes = nil
	file_multi_v1_channel_proto_depIdxs = nil
}
