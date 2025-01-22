/*
Copyright (c) 2025 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConditionalRequiredValidator struct {
	DependentFieldName string
	ExpectedValue      string
}

func (v ConditionalRequiredValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Ensures a value is set if '%s' equals '%s'.", v.DependentFieldName, v.ExpectedValue)
}

func (v ConditionalRequiredValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Ensures a value is set if **%s** equals '%s'.", v.DependentFieldName, v.ExpectedValue)
}

func (v ConditionalRequiredValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {

	var dependentFieldValue types.String
	diags := req.Config.GetAttribute(ctx, path.Root(v.DependentFieldName), &dependentFieldValue)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if dependentFieldValue.IsUnknown() || dependentFieldValue.ValueString() != v.ExpectedValue {
		return
	}

	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Validation Error",
			fmt.Sprintf("Field '%s' is required when '%s' equals '%s'.", req.Path.String(), v.DependentFieldName, v.ExpectedValue),
		)
	}
}

func ChangeToRequired(dependentFieldName, expectedValue string) validator.String {
	return ConditionalRequiredValidator{
		DependentFieldName: dependentFieldName,
		ExpectedValue:      expectedValue,
	}
}
