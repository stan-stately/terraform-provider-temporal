package validators

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type StringDurationValidator struct {
	AllowsEmpty bool
}

func (v StringDurationValidator) Description(ctx context.Context) string {
	return "Ensures the string represents a duration."
}

func (v StringDurationValidator) MarkdownDescription(ctx context.Context) string {
	return "Ensures the string represents a duration. E.g 30s, 5m, 10h or 8d."
}

func (v StringDurationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	attr, _ := req.Config.Schema.AttributeAtPath(ctx, req.Path)
	if (attr.IsOptional() && req.ConfigValue.IsNull()) || req.ConfigValue.IsUnknown() {
		return
	}
	durationStr := req.ConfigValue.ValueString()
	if len(durationStr) >= 2 {
		unit := string(durationStr[len(durationStr)-1])
		valueStr := durationStr[:len(durationStr)-1]
		_, err := strconv.Atoi(valueStr)
		if err == nil {
			if unit == "s" || unit == "m" || unit == "h" || unit == "d" {
				return
			}
		}
	}
	resp.Diagnostics.AddError(
		"Invalid Value at "+req.PathExpression.String(),
		"The value must represent a duration. E.g 30s, 5m, 10h or 8d.",
	)
}
