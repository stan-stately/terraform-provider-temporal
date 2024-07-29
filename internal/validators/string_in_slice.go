package validators

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"strings"
)

type StringInSliceValidator struct {
	AllowedValues []string
}

func (v StringInSliceValidator) Description(ctx context.Context) string {
	return "Ensures the string is one of the allowed values."
}

func (v StringInSliceValidator) MarkdownDescription(ctx context.Context) string {
	return "Ensures the string is one of the allowed values: `yes`, `no`, `maybe`."
}

func (v StringInSliceValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for _, value := range v.AllowedValues {
		if req.ConfigValue.ValueString() == value {
			return
		}
	}

	resp.Diagnostics.AddError(
		"Invalid Value",
		"The value must be one of: "+strings.Join(v.AllowedValues[:len(v.AllowedValues)-1], ", ")+" or "+v.AllowedValues[len(v.AllowedValues)-1],
	)
}
