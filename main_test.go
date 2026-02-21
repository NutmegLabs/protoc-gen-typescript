package main

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// resetMaps reinitializes the global messageMap and enumMap between tests.
func resetMaps() {
	messageMap = make(map[string]*Message)
	enumMap = make(map[string]*Enum)
}

func makeField(name string, typ descriptor.FieldDescriptorProto_Type, label descriptor.FieldDescriptorProto_Label) *descriptor.FieldDescriptorProto {
	return &descriptor.FieldDescriptorProto{
		Name:  proto.String(name),
		Type:  typ.Enum(),
		Label: label.Enum(),
	}
}

func makeTypedField(name string, typ descriptor.FieldDescriptorProto_Type, label descriptor.FieldDescriptorProto_Label, typeName string) *descriptor.FieldDescriptorProto {
	f := makeField(name, typ, label)
	f.TypeName = proto.String(typeName)
	return f
}

func makeEnumValue(name string, number int32) *descriptor.EnumValueDescriptorProto {
	return &descriptor.EnumValueDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(number),
	}
}

func TestShouldEmitFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"wrappers included", "google/protobuf/wrappers.proto", true},
		{"descriptor excluded", "google/protobuf/descriptor.proto", false},
		{"timestamp excluded", "google/protobuf/timestamp.proto", false},
		{"custom file included", "mypackage/myfile.proto", true},
		{"empty string included", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldEmitFile(tt.filename); got != tt.want {
				t.Errorf("shouldEmitFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestGetScopedName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"package and message", ".mypackage.MyMessage", "mypackage$MyMessage"},
		{"deeply nested", ".a.b.c.D", "a$b$c$D"},
		{"no dots", "NoDots", "any"},
		{"single component", ".single", "single"},
		{"empty string", "", "any"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getScopedName(tt.input); got != tt.want {
				t.Errorf("getScopedName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetFieldType(t *testing.T) {
	optional := descriptor.FieldDescriptorProto_LABEL_OPTIONAL
	repeated := descriptor.FieldDescriptorProto_LABEL_REPEATED

	t.Run("numeric types", func(t *testing.T) {
		resetMaps()
		numericTypes := []descriptor.FieldDescriptorProto_Type{
			descriptor.FieldDescriptorProto_TYPE_DOUBLE,
			descriptor.FieldDescriptorProto_TYPE_FLOAT,
			descriptor.FieldDescriptorProto_TYPE_INT32,
			descriptor.FieldDescriptorProto_TYPE_FIXED32,
			descriptor.FieldDescriptorProto_TYPE_UINT32,
			descriptor.FieldDescriptorProto_TYPE_SFIXED32,
			descriptor.FieldDescriptorProto_TYPE_SINT32,
		}
		for _, typ := range numericTypes {
			f := makeField("test", typ, optional)
			got := getFieldType("ns", f)
			if got != "number" {
				t.Errorf("getFieldType for %v = %q, want %q", typ, got, "number")
			}
		}
	})

	t.Run("64-bit integers", func(t *testing.T) {
		resetMaps()
		int64Types := []descriptor.FieldDescriptorProto_Type{
			descriptor.FieldDescriptorProto_TYPE_INT64,
			descriptor.FieldDescriptorProto_TYPE_UINT64,
			descriptor.FieldDescriptorProto_TYPE_FIXED64,
			descriptor.FieldDescriptorProto_TYPE_SFIXED64,
			descriptor.FieldDescriptorProto_TYPE_SINT64,
		}
		for _, typ := range int64Types {
			f := makeField("test", typ, optional)
			got := getFieldType("ns", f)
			if got != "string" {
				t.Errorf("getFieldType for %v = %q, want %q", typ, got, "string")
			}
		}
	})

	t.Run("bool string bytes", func(t *testing.T) {
		resetMaps()
		cases := []struct {
			typ  descriptor.FieldDescriptorProto_Type
			want string
		}{
			{descriptor.FieldDescriptorProto_TYPE_BOOL, "boolean"},
			{descriptor.FieldDescriptorProto_TYPE_STRING, "string"},
			{descriptor.FieldDescriptorProto_TYPE_BYTES, "any"},
		}
		for _, tc := range cases {
			f := makeField("test", tc.typ, optional)
			got := getFieldType("ns", f)
			if got != tc.want {
				t.Errorf("getFieldType for %v = %q, want %q", tc.typ, got, tc.want)
			}
		}
	})

	t.Run("repeated field", func(t *testing.T) {
		resetMaps()
		f := makeField("items", descriptor.FieldDescriptorProto_TYPE_STRING, repeated)
		got := getFieldType("ns", f)
		if got != "string[]" {
			t.Errorf("getFieldType = %q, want %q", got, "string[]")
		}
	})

	t.Run("enum field", func(t *testing.T) {
		resetMaps()
		enumMap["pkg$Status"] = &Enum{Name: "pkg$Status", Values: []string{"ACTIVE"}}
		f := makeTypedField("status", descriptor.FieldDescriptorProto_TYPE_ENUM, optional, ".pkg.Status")
		got := getFieldType("ns", f)
		if got != "pkg$Status" {
			t.Errorf("getFieldType = %q, want %q", got, "pkg$Status")
		}
	})

	t.Run("message field", func(t *testing.T) {
		resetMaps()
		messageMap["pkg$Address"] = &Message{Name: "pkg$Address", Fields: []*Field{{Name: "street", Type: "string"}}}
		f := makeTypedField("address", descriptor.FieldDescriptorProto_TYPE_MESSAGE, optional, ".pkg.Address")
		got := getFieldType("ns", f)
		if got != "pkg$Address" {
			t.Errorf("getFieldType = %q, want %q", got, "pkg$Address")
		}
	})

	t.Run("timestamp", func(t *testing.T) {
		resetMaps()
		f := makeTypedField("created_at", descriptor.FieldDescriptorProto_TYPE_MESSAGE, optional, ".google.protobuf.Timestamp")
		got := getFieldType("ns", f)
		if got != "string" {
			t.Errorf("getFieldType = %q, want %q", got, "string")
		}
	})

	t.Run("map field", func(t *testing.T) {
		resetMaps()
		messageMap["pkg$MyMsg$LabelsEntry"] = &Message{
			Name: "pkg$MyMsg$LabelsEntry",
			Fields: []*Field{
				{Name: "key", Type: "string"},
				{Name: "value", Type: "number"},
			},
			IsMap: true,
		}
		f := makeTypedField("labels", descriptor.FieldDescriptorProto_TYPE_MESSAGE, repeated, ".pkg.MyMsg.LabelsEntry")
		got := getFieldType("ns", f)
		if got != "{ [key: string]: number }" {
			t.Errorf("getFieldType = %q, want %q", got, "{ [key: string]: number }")
		}
	})
}

func TestRegisterAndCollectEnum(t *testing.T) {
	resetMaps()

	enum := &descriptor.EnumDescriptorProto{
		Name: proto.String("Status"),
		Value: []*descriptor.EnumValueDescriptorProto{
			makeEnumValue("STATUS_UNKNOWN", 0),
			makeEnumValue("STATUS_ACTIVE", 1),
			makeEnumValue("STATUS_INACTIVE", 2),
		},
	}

	registerEnumType("pkg$", enum)

	e := enumMap["pkg$Status"]
	if e == nil {
		t.Fatal("enum not found in enumMap")
	}
	if len(e.Values) != 2 {
		t.Fatalf("got %d values, want 2 (zero-value should be skipped)", len(e.Values))
	}
	if e.Values[0] != "STATUS_ACTIVE" || e.Values[1] != "STATUS_INACTIVE" {
		t.Errorf("values = %v, want [STATUS_ACTIVE STATUS_INACTIVE]", e.Values)
	}

	entries := collectEnumEntries("pkg$", enum)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].name != "pkg$Status" {
		t.Errorf("name = %q, want %q", entries[0].name, "pkg$Status")
	}

	expected := `
export type pkg$Status =
	| 'STATUS_ACTIVE'
	| 'STATUS_INACTIVE'
;
`
	if entries[0].output != expected {
		t.Errorf("output = %q, want %q", entries[0].output, expected)
	}
}

func TestRegisterAndCollectMessage(t *testing.T) {
	resetMaps()

	msg := &descriptor.DescriptorProto{
		Name: proto.String("Person"),
		Field: []*descriptor.FieldDescriptorProto{
			makeField("name", descriptor.FieldDescriptorProto_TYPE_STRING, descriptor.FieldDescriptorProto_LABEL_OPTIONAL),
			makeField("age", descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_LABEL_OPTIONAL),
		},
	}

	registerMessageType("pkg$", msg)

	entries := collectMessageEntries("pkg$", msg)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].name != "pkg$Person" {
		t.Errorf("name = %q, want %q", entries[0].name, "pkg$Person")
	}

	out := entries[0].output
	if !strings.Contains(out, "export type pkg$Person = {") {
		t.Errorf("output missing type declaration: %q", out)
	}
	if !strings.Contains(out, "name?: string,") {
		t.Errorf("output missing name field: %q", out)
	}
	if !strings.Contains(out, "age?: number,") {
		t.Errorf("output missing age field: %q", out)
	}
	if !strings.Contains(out, "};") {
		t.Errorf("output missing closing brace: %q", out)
	}
}

func TestRegisterAndCollectMessageWithNestedTypes(t *testing.T) {
	resetMaps()

	msg := &descriptor.DescriptorProto{
		Name: proto.String("Outer"),
		Field: []*descriptor.FieldDescriptorProto{
			makeField("name", descriptor.FieldDescriptorProto_TYPE_STRING, descriptor.FieldDescriptorProto_LABEL_OPTIONAL),
			makeTypedField("status", descriptor.FieldDescriptorProto_TYPE_ENUM, descriptor.FieldDescriptorProto_LABEL_OPTIONAL, ".pkg.Outer.Status"),
			makeTypedField("inner", descriptor.FieldDescriptorProto_TYPE_MESSAGE, descriptor.FieldDescriptorProto_LABEL_OPTIONAL, ".pkg.Outer.Inner"),
		},
		EnumType: []*descriptor.EnumDescriptorProto{
			{
				Name: proto.String("Status"),
				Value: []*descriptor.EnumValueDescriptorProto{
					makeEnumValue("UNKNOWN", 0),
					makeEnumValue("ACTIVE", 1),
				},
			},
		},
		NestedType: []*descriptor.DescriptorProto{
			{
				Name: proto.String("Inner"),
				Field: []*descriptor.FieldDescriptorProto{
					makeField("value", descriptor.FieldDescriptorProto_TYPE_INT32, descriptor.FieldDescriptorProto_LABEL_OPTIONAL),
				},
			},
		},
	}

	registerMessageType("pkg$", msg)

	entries := collectMessageEntries("pkg$", msg)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	// collectMessageEntries processes: nested enums, nested messages, then outer message
	if entries[0].name != "pkg$Outer$Status" {
		t.Errorf("entries[0].name = %q, want %q", entries[0].name, "pkg$Outer$Status")
	}
	if entries[1].name != "pkg$Outer$Inner" {
		t.Errorf("entries[1].name = %q, want %q", entries[1].name, "pkg$Outer$Inner")
	}
	if entries[2].name != "pkg$Outer" {
		t.Errorf("entries[2].name = %q, want %q", entries[2].name, "pkg$Outer")
	}

	// Verify nested enum output
	if !strings.Contains(entries[0].output, "export type pkg$Outer$Status =") {
		t.Errorf("nested enum output missing declaration: %q", entries[0].output)
	}
	if !strings.Contains(entries[0].output, "| 'ACTIVE'") {
		t.Errorf("nested enum output missing value: %q", entries[0].output)
	}

	// Verify nested message output
	if !strings.Contains(entries[1].output, "export type pkg$Outer$Inner = {") {
		t.Errorf("nested message output missing declaration: %q", entries[1].output)
	}

	// Verify outer message references nested types
	if !strings.Contains(entries[2].output, "status?: pkg$Outer$Status,") {
		t.Errorf("outer message missing enum field: %q", entries[2].output)
	}
	if !strings.Contains(entries[2].output, "inner?: pkg$Outer$Inner,") {
		t.Errorf("outer message missing nested message field: %q", entries[2].output)
	}
}

func TestCollectServiceEntries(t *testing.T) {
	resetMaps()

	svc := &descriptor.ServiceDescriptorProto{
		Name: proto.String("Greeter"),
		Method: []*descriptor.MethodDescriptorProto{
			{
				Name:       proto.String("SayHello"),
				InputType:  proto.String(".pkg.HelloRequest"),
				OutputType: proto.String(".pkg.HelloResponse"),
			},
		},
	}

	entries := collectServiceEntries("pkg$", svc)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].name != "pkg$Greeter" {
		t.Errorf("name = %q, want %q", entries[0].name, "pkg$Greeter")
	}

	out := entries[0].output
	if !strings.Contains(out, "export interface pkg$Greeter") {
		t.Errorf("output missing interface declaration: %q", out)
	}
	if !strings.Contains(out, "SayHello(request: pkg$HelloRequest") {
		t.Errorf("output missing method with request type: %q", out)
	}
	if !strings.Contains(out, "response: pkg$HelloResponse") {
		t.Errorf("output missing response type: %q", out)
	}
}
