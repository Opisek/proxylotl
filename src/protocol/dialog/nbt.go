package dialog

import (
	"bytes"
	util "proxelot/protocol/internal"
	"reflect"
	"strings"
)

const (
	tag_end = iota
	tag_byte
	tag_short
	tag_int
	tag_long
	tag_float
	tag_double
	tag_byte_array
	tag_string
	tag_list
	tag_compound
	tag_int_array
	tag_long_array
)

func serializeNbt(nbt any) []byte {
	var buf bytes.Buffer
	_recursivelySerialize(nbt, "", false, &buf)
	return buf.Bytes()
}

func _recursivelySerialize(nbt any, compoundName string, omitType bool, buf *bytes.Buffer) {
	v := reflect.ValueOf(nbt)

	switch v.Kind() {

	// Dereference pointers
	case reflect.Pointer:
		_recursivelySerialize(v.Elem().Interface(), compoundName, false, buf)

	// Iterate over all fields in a struct
	case reflect.Struct:
		if !omitType {
			buf.Write(util.SerializeVarInt(tag_compound))
		}

		if compoundName != "" {
			buf.Write(util.SerializeNbtString(compoundName))
		}

		for i := range v.NumField() {
			childName := v.Type().Field(i).Tag.Get("nbt")
			if childName == "" { // If empty then top compound => No name in network NBT
				childName = strings.ToLower(v.Type().Field(i).Name)
			}
			_recursivelySerialize(v.Field(i).Interface(), childName, false, buf)
		}

		buf.Write(util.SerializeVarInt(tag_end))

	// Serialize string
	case reflect.String:
		if !omitType {
			buf.Write(util.SerializeVarInt(tag_string))
		}

		if compoundName != "" {
			buf.Write(util.SerializeNbtString(compoundName))
		}

		buf.Write(util.SerializeNbtString(nbt.(string)))

	// Seralize uint8
	case reflect.Uint8:
		if !omitType {
			buf.Write(util.SerializeVarInt(tag_byte))
		}

		if compoundName != "" {
			buf.Write(util.SerializeNbtString(compoundName))
		}

		buf.WriteByte(nbt.(byte))

	// Seralize uint16
	case reflect.Uint16:
		if !omitType {
			buf.Write(util.SerializeVarInt(tag_short))
		}

		if compoundName != "" {
			buf.Write(util.SerializeNbtString(compoundName))
		}

		buf.Write(util.SerializeUnsignedShort(nbt.(uint16)))

	// Seralize uint32
	case reflect.Uint32:
		if !omitType {
			buf.Write(util.SerializeVarInt(tag_int))
		}

		if compoundName != "" {
			buf.Write(util.SerializeNbtString(compoundName))
		}

		buf.Write(util.SerializeUnsignedInt(nbt.(uint32)))

	// Seralize uint64
	case reflect.Uint64:
		if !omitType {
			buf.Write(util.SerializeVarInt(tag_long))
		}

		if compoundName != "" {
			buf.Write(util.SerializeNbtString(compoundName))
		}

		buf.Write(util.SerializeUnsignedLong(nbt.(uint64)))

	// Iterate over arrays
	case reflect.Slice:
		len := v.Len()
		arrType := reflect.TypeOf(nbt).Elem().Kind()

		switch arrType {
		case reflect.Uint8:
			if !omitType {
				buf.Write(util.SerializeVarInt(tag_byte_array))
			}
			if compoundName != "" {
				buf.Write(util.SerializeNbtString(compoundName))
			}
			buf.Write(util.SerializeUnsignedInt(uint32(len)))

		case reflect.Uint32:
			if !omitType {
				buf.Write(util.SerializeVarInt(tag_int_array))
			}
			if compoundName != "" {
				buf.Write(util.SerializeNbtString(compoundName))
			}
			buf.Write(util.SerializeUnsignedInt(uint32(len)))

		case reflect.Uint64:
			if !omitType {
				buf.Write(util.SerializeVarInt(tag_long_array))
			}
			if compoundName != "" {
				buf.Write(util.SerializeNbtString(compoundName))
			}

		default:
			if !omitType {
				buf.Write(util.SerializeVarInt(tag_list))
			}
			if compoundName != "" {
				buf.Write(util.SerializeNbtString(compoundName))
			}
			if len == 0 {
				buf.WriteByte(tag_end)
			} else if arrType == reflect.Uint16 {
				buf.WriteByte(tag_short)
			} else if arrType == reflect.Slice {
				buf.WriteByte(tag_list)
			} else if arrType == reflect.String {
				buf.WriteByte(tag_string)
			} else if arrType == reflect.Struct || arrType == reflect.Interface {
				buf.WriteByte(tag_compound)
			} else {
				panic("not implemented")
			}
		}

		buf.Write(util.SerializeUnsignedInt(uint32(len)))
		for i := range len {
			_recursivelySerialize(v.Index(i).Interface(), "", true, buf)
		}

	case reflect.Float32:
		panic("not implemented")

	case reflect.Float64:
		panic("not implemented")

	default:
		panic("not implemented")
	}
}
