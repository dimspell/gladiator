// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: multi/v1/character.proto

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

type GetCharacterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId        int64  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	CharacterName string `protobuf:"bytes,2,opt,name=character_name,json=characterName,proto3" json:"character_name,omitempty"`
}

func (x *GetCharacterRequest) Reset() {
	*x = GetCharacterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCharacterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCharacterRequest) ProtoMessage() {}

func (x *GetCharacterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCharacterRequest.ProtoReflect.Descriptor instead.
func (*GetCharacterRequest) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{0}
}

func (x *GetCharacterRequest) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *GetCharacterRequest) GetCharacterName() string {
	if x != nil {
		return x.CharacterName
	}
	return ""
}

type GetCharacterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Character *Character `protobuf:"bytes,1,opt,name=character,proto3" json:"character,omitempty"`
}

func (x *GetCharacterResponse) Reset() {
	*x = GetCharacterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetCharacterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetCharacterResponse) ProtoMessage() {}

func (x *GetCharacterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetCharacterResponse.ProtoReflect.Descriptor instead.
func (*GetCharacterResponse) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{1}
}

func (x *GetCharacterResponse) GetCharacter() *Character {
	if x != nil {
		return x.Character
	}
	return nil
}

type ListCharactersRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId int64 `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
}

func (x *ListCharactersRequest) Reset() {
	*x = ListCharactersRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListCharactersRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListCharactersRequest) ProtoMessage() {}

func (x *ListCharactersRequest) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListCharactersRequest.ProtoReflect.Descriptor instead.
func (*ListCharactersRequest) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{2}
}

func (x *ListCharactersRequest) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

type ListCharactersResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Characters []*Character `protobuf:"bytes,1,rep,name=characters,proto3" json:"characters,omitempty"`
}

func (x *ListCharactersResponse) Reset() {
	*x = ListCharactersResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListCharactersResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListCharactersResponse) ProtoMessage() {}

func (x *ListCharactersResponse) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListCharactersResponse.ProtoReflect.Descriptor instead.
func (*ListCharactersResponse) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{3}
}

func (x *ListCharactersResponse) GetCharacters() []*Character {
	if x != nil {
		return x.Characters
	}
	return nil
}

type DeleteCharacterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId        int64  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	CharacterName string `protobuf:"bytes,2,opt,name=character_name,json=characterName,proto3" json:"character_name,omitempty"`
}

func (x *DeleteCharacterRequest) Reset() {
	*x = DeleteCharacterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteCharacterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteCharacterRequest) ProtoMessage() {}

func (x *DeleteCharacterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteCharacterRequest.ProtoReflect.Descriptor instead.
func (*DeleteCharacterRequest) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{4}
}

func (x *DeleteCharacterRequest) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *DeleteCharacterRequest) GetCharacterName() string {
	if x != nil {
		return x.CharacterName
	}
	return ""
}

type DeleteCharacterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DeleteCharacterResponse) Reset() {
	*x = DeleteCharacterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteCharacterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteCharacterResponse) ProtoMessage() {}

func (x *DeleteCharacterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteCharacterResponse.ProtoReflect.Descriptor instead.
func (*DeleteCharacterResponse) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{5}
}

type CreateCharacterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId        int64  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	CharacterName string `protobuf:"bytes,2,opt,name=character_name,json=characterName,proto3" json:"character_name,omitempty"`
	Info          []byte `protobuf:"bytes,3,opt,name=info,proto3" json:"info,omitempty"`
}

func (x *CreateCharacterRequest) Reset() {
	*x = CreateCharacterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateCharacterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateCharacterRequest) ProtoMessage() {}

func (x *CreateCharacterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateCharacterRequest.ProtoReflect.Descriptor instead.
func (*CreateCharacterRequest) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{6}
}

func (x *CreateCharacterRequest) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *CreateCharacterRequest) GetCharacterName() string {
	if x != nil {
		return x.CharacterName
	}
	return ""
}

func (x *CreateCharacterRequest) GetInfo() []byte {
	if x != nil {
		return x.Info
	}
	return nil
}

type CreateCharacterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Character *Character `protobuf:"bytes,1,opt,name=character,proto3" json:"character,omitempty"`
}

func (x *CreateCharacterResponse) Reset() {
	*x = CreateCharacterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_multi_v1_character_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateCharacterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateCharacterResponse) ProtoMessage() {}

func (x *CreateCharacterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_multi_v1_character_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateCharacterResponse.ProtoReflect.Descriptor instead.
func (*CreateCharacterResponse) Descriptor() ([]byte, []int) {
	return file_multi_v1_character_proto_rawDescGZIP(), []int{7}
}

func (x *CreateCharacterResponse) GetCharacter() *Character {
	if x != nil {
		return x.Character
	}
	return nil
}

var File_multi_v1_character_proto protoreflect.FileDescriptor

var file_multi_v1_character_proto_rawDesc = []byte{
	0x0a, 0x18, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x6d, 0x75, 0x6c, 0x74,
	0x69, 0x2e, 0x76, 0x31, 0x1a, 0x1d, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63,
	0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x55, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73,
	0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x75, 0x73, 0x65,
	0x72, 0x49, 0x64, 0x12, 0x25, 0x0a, 0x0e, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x63, 0x68, 0x61,
	0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x49, 0x0a, 0x14, 0x47, 0x65,
	0x74, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x31, 0x0a, 0x09, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x09, 0x63, 0x68, 0x61, 0x72,
	0x61, 0x63, 0x74, 0x65, 0x72, 0x22, 0x30, 0x0a, 0x15, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x68, 0x61,
	0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17,
	0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x22, 0x4d, 0x0a, 0x16, 0x4c, 0x69, 0x73, 0x74, 0x43,
	0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x33, 0x0a, 0x0a, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x0a, 0x63, 0x68, 0x61, 0x72,
	0x61, 0x63, 0x74, 0x65, 0x72, 0x73, 0x22, 0x58, 0x0a, 0x16, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x25, 0x0a, 0x0e, 0x63, 0x68, 0x61,
	0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0d, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65,
	0x22, 0x19, 0x0a, 0x17, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x6c, 0x0a, 0x16, 0x43,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x25,
	0x0a, 0x0e, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x5f, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x04, 0x69, 0x6e, 0x66, 0x6f, 0x22, 0x4c, 0x0a, 0x17, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x31, 0x0a, 0x09, 0x63, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e,
	0x76, 0x31, 0x2e, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x09, 0x63, 0x68,
	0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x32, 0xee, 0x02, 0x0a, 0x10, 0x43, 0x68, 0x61, 0x72,
	0x61, 0x63, 0x74, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x4f, 0x0a, 0x0c,
	0x47, 0x65, 0x74, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x1d, 0x2e, 0x6d,
	0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1e, 0x2e, 0x6d, 0x75,
	0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63,
	0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x55, 0x0a,
	0x0e, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x73, 0x12,
	0x1f, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x43,
	0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x20, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74,
	0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x00, 0x12, 0x58, 0x0a, 0x0f, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x68,
	0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x20, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e,
	0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74,
	0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x21, 0x2e, 0x6d, 0x75, 0x6c, 0x74,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x58,
	0x0a, 0x0f, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65,
	0x72, 0x12, 0x20, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65, 0x6c,
	0x65, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x21, 0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x44,
	0x65, 0x6c, 0x65, 0x74, 0x65, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x97, 0x01, 0x0a, 0x0c, 0x63, 0x6f, 0x6d,
	0x2e, 0x6d, 0x75, 0x6c, 0x74, 0x69, 0x2e, 0x76, 0x31, 0x42, 0x0e, 0x43, 0x68, 0x61, 0x72, 0x61,
	0x63, 0x74, 0x65, 0x72, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x36, 0x67, 0x69, 0x74,
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
	file_multi_v1_character_proto_rawDescOnce sync.Once
	file_multi_v1_character_proto_rawDescData = file_multi_v1_character_proto_rawDesc
)

func file_multi_v1_character_proto_rawDescGZIP() []byte {
	file_multi_v1_character_proto_rawDescOnce.Do(func() {
		file_multi_v1_character_proto_rawDescData = protoimpl.X.CompressGZIP(file_multi_v1_character_proto_rawDescData)
	})
	return file_multi_v1_character_proto_rawDescData
}

var file_multi_v1_character_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_multi_v1_character_proto_goTypes = []interface{}{
	(*GetCharacterRequest)(nil),     // 0: multi.v1.GetCharacterRequest
	(*GetCharacterResponse)(nil),    // 1: multi.v1.GetCharacterResponse
	(*ListCharactersRequest)(nil),   // 2: multi.v1.ListCharactersRequest
	(*ListCharactersResponse)(nil),  // 3: multi.v1.ListCharactersResponse
	(*DeleteCharacterRequest)(nil),  // 4: multi.v1.DeleteCharacterRequest
	(*DeleteCharacterResponse)(nil), // 5: multi.v1.DeleteCharacterResponse
	(*CreateCharacterRequest)(nil),  // 6: multi.v1.CreateCharacterRequest
	(*CreateCharacterResponse)(nil), // 7: multi.v1.CreateCharacterResponse
	(*Character)(nil),               // 8: multi.v1.Character
}
var file_multi_v1_character_proto_depIdxs = []int32{
	8, // 0: multi.v1.GetCharacterResponse.character:type_name -> multi.v1.Character
	8, // 1: multi.v1.ListCharactersResponse.characters:type_name -> multi.v1.Character
	8, // 2: multi.v1.CreateCharacterResponse.character:type_name -> multi.v1.Character
	0, // 3: multi.v1.CharacterService.GetCharacter:input_type -> multi.v1.GetCharacterRequest
	2, // 4: multi.v1.CharacterService.ListCharacters:input_type -> multi.v1.ListCharactersRequest
	6, // 5: multi.v1.CharacterService.CreateCharacter:input_type -> multi.v1.CreateCharacterRequest
	4, // 6: multi.v1.CharacterService.DeleteCharacter:input_type -> multi.v1.DeleteCharacterRequest
	1, // 7: multi.v1.CharacterService.GetCharacter:output_type -> multi.v1.GetCharacterResponse
	3, // 8: multi.v1.CharacterService.ListCharacters:output_type -> multi.v1.ListCharactersResponse
	7, // 9: multi.v1.CharacterService.CreateCharacter:output_type -> multi.v1.CreateCharacterResponse
	5, // 10: multi.v1.CharacterService.DeleteCharacter:output_type -> multi.v1.DeleteCharacterResponse
	7, // [7:11] is the sub-list for method output_type
	3, // [3:7] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_multi_v1_character_proto_init() }
func file_multi_v1_character_proto_init() {
	if File_multi_v1_character_proto != nil {
		return
	}
	file_multi_v1_character_type_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_multi_v1_character_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCharacterRequest); i {
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
		file_multi_v1_character_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetCharacterResponse); i {
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
		file_multi_v1_character_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListCharactersRequest); i {
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
		file_multi_v1_character_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListCharactersResponse); i {
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
		file_multi_v1_character_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteCharacterRequest); i {
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
		file_multi_v1_character_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteCharacterResponse); i {
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
		file_multi_v1_character_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateCharacterRequest); i {
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
		file_multi_v1_character_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateCharacterResponse); i {
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
			RawDescriptor: file_multi_v1_character_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_multi_v1_character_proto_goTypes,
		DependencyIndexes: file_multi_v1_character_proto_depIdxs,
		MessageInfos:      file_multi_v1_character_proto_msgTypes,
	}.Build()
	File_multi_v1_character_proto = out.File
	file_multi_v1_character_proto_rawDesc = nil
	file_multi_v1_character_proto_goTypes = nil
	file_multi_v1_character_proto_depIdxs = nil
}
