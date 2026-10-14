package keatypes

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/wfdewith/terraform-provider-kea/kea"
)

var (
	_ basetypes.StringValuable                   = (*ByteStringValue)(nil)
	_ basetypes.StringValuableWithSemanticEquals = (*ByteStringValue)(nil)
	_ xattr.ValidateableAttribute                = (*ByteStringValue)(nil)
	_ function.ValidateableParameter             = (*ByteStringValue)(nil)
)

type ByteStringValue struct {
	basetypes.StringValue
}
type ByteString = ByteStringValue

func (v ByteStringValue) Equal(o attr.Value) bool {
	other, ok := o.(ByteStringValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v ByteStringValue) Type(ctx context.Context) attr.Type {
	return ByteStringType{}
}

func (v ByteStringValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(ByteStringValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Expected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)
		return false, diags
	}

	// Already validated at this point, ignoring errors
	newByteString, _ := kea.ParseByteString(newValue.ValueString())
	currentByteString, _ := kea.ParseByteString(v.ValueString())

	return bytes.Equal(currentByteString, newByteString), diags
}

func (v ByteStringValue) ValidateAttribute(_ context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
	if v.IsUnknown() || v.IsNull() {
		return
	}
	_, err := kea.ParseByteString(v.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Byte String Value",
			"A string value was provided that is not valid byte string format.\n\n"+
				"Given Value: "+v.ValueString()+"\n"+
				"Error: "+err.Error(),
		)
	}
}

func (v ByteStringValue) ValidateParameter(ctx context.Context, req function.ValidateParameterRequest, resp *function.ValidateParameterResponse) {
	if v.IsUnknown() || v.IsNull() {
		return
	}
	_, err := kea.ParseByteString(v.ValueString())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(
			req.Position,
			"Invalid Byte String Value: "+
				"A string value was provided that is not valid byte string format.\n\n"+
				fmt.Sprintf("Position: %d", req.Position)+"\n"+
				"Given Value: "+v.ValueString()+"\n"+
				"Error: "+err.Error(),
		)
	}
}

func (v ByteStringValue) ValueByteString() (kea.ByteString, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v.IsNull() {
		diags.AddError("ByteString ValueByteString Error", "byte string value is null")
		return nil, diags
	}

	if v.IsUnknown() {
		diags.AddError("ByteString ValueByteString Error", "byte string value is unknown")
		return nil, diags
	}

	byteString, err := kea.ParseByteString(v.ValueString())
	if err != nil {
		diags.AddError("ByteString ValueByteString Error", err.Error())
		return nil, diags
	}

	return byteString, nil
}

func NewByteStringNull() ByteStringValue {
	return ByteStringValue{
		StringValue: basetypes.NewStringNull(),
	}
}

func NewByteStringUnknown() ByteStringValue {
	return ByteStringValue{
		StringValue: basetypes.NewStringUnknown(),
	}
}

func NewByteStringValue(value string) ByteStringValue {
	return ByteStringValue{
		StringValue: basetypes.NewStringValue(value),
	}
}

func NewByteStringPointerValue(value *string) ByteStringValue {
	return ByteStringValue{
		StringValue: basetypes.NewStringPointerValue(value),
	}
}
