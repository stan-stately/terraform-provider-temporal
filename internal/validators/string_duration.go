package validators

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// StringDurationValidator struct is now empty as AllowsEmpty was unused.
type StringDurationValidator struct{}

func (v StringDurationValidator) Description(ctx context.Context) string {
	return "Ensures the string represents a valid Go duration."
}

func (v StringDurationValidator) MarkdownDescription(ctx context.Context) string {
	return "Ensures the string represents a valid Go duration. For example: `30s`, `5m`, `10h`."
}

func (v StringDurationValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {

	attr, _ := req.Config.Schema.AttributeAtPath(ctx, req.Path)
	if (attr.IsOptional() && req.ConfigValue.IsNull()) || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Use Go's standard library to parse the duration.
	// This is more robust and handles more formats than manual parsing.
	_, err := time.ParseDuration(value)
	if err != nil {
		// Use AddAttributeError for more specific error messages tied to the attribute.
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration String",
			fmt.Sprintf("The value '%s' is not a valid duration. Please use a format like '30s', '1.5h', or '10m'. Original error: %s", value, err),
		)
	}
}
