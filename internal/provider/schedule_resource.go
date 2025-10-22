package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"terraform-provider-temporal/internal/validators"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	temporal "go.temporal.io/sdk/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &scheduleResource{}
	_ resource.ResourceWithConfigure = &scheduleResource{}
)

func NewScheduleResource() resource.Resource {
	return &scheduleResource{}
}

type scheduleResource struct {
	client    temporal.Client
	namespace string
}

type scheduleActionModel struct {
	InputPayload  basetypes.StringValue `tfsdk:"input_payload"`
	WorkflowId    basetypes.StringValue `tfsdk:"workflow_id"`
	WorkflowType  basetypes.StringValue `tfsdk:"workflow_type"`
	TaskQueueName basetypes.StringValue `tfsdk:"task_queue_name"`
}

type scheduleIntervalModel struct {
	Every  basetypes.StringValue `tfsdk:"every"`
	Offset basetypes.StringValue `tfsdk:"offset"`
}

type scheduleSpecModel struct {
	Intervals []scheduleIntervalModel `tfsdk:"interval"`
}

type scheduleResourceModel struct {
	Name           basetypes.StringValue `tfsdk:"name"`
	IsPaused       basetypes.BoolValue   `tfsdk:"is_paused"`
	Action         scheduleActionModel   `tfsdk:"action"`
	OverlapPolicy  basetypes.StringValue `tfsdk:"overlap_policy"`
	CatchupWindow  basetypes.StringValue `tfsdk:"catchup_window"`
	PauseOnFailure basetypes.BoolValue   `tfsdk:"pause_on_failure"`
	Spec           scheduleSpecModel     `tfsdk:"spec"`
}

func stringToScheduleOverlapPolicy(v string) enums.ScheduleOverlapPolicy {
	switch v {
	case "skip":
		return enums.SCHEDULE_OVERLAP_POLICY_SKIP
	case "buffer_one":
		return enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE
	case "buffer_all":
		return enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL
	case "cancel_other":
		return enums.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER
	case "terminate_other":

		return enums.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER
	case "allow_all":
		return enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL
	default:
		return enums.SCHEDULE_OVERLAP_POLICY_UNSPECIFIED
	}
}
func scheduleOverlapPolicyToString(p enums.ScheduleOverlapPolicy) string {
	switch p {
	case enums.SCHEDULE_OVERLAP_POLICY_SKIP:
		return "skip"
	case enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE:
		return "buffer_one"
	case enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL:
		return "buffer_all"
	case enums.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER:
		return "cancel_other"
	case enums.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER:
		return "terminate_other"
	case enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL:
		return "allow_all"
	}
	return "unspecified"
}

func parseScheduleResource(name string, response *workflowservice.DescribeScheduleResponse) *scheduleResourceModel {
	actionDetails := response.GetSchedule().GetAction().GetStartWorkflow()

	intervals := make([]scheduleIntervalModel, 0)
	for _, i := range response.GetSchedule().GetSpec().GetInterval() {
		intervals = append(intervals, scheduleIntervalModel{
			Every:  types.StringValue(formatDuration(i.GetInterval().AsDuration())),
			Offset: types.StringValue(formatDuration(i.GetPhase().AsDuration())),
		})
	}

	// Sort intervals to ensure consistent ordering for set comparison
	sort.Slice(intervals, func(i, j int) bool {
		// Sort by every duration first, then by offset
		if intervals[i].Every.ValueString() != intervals[j].Every.ValueString() {
			return intervals[i].Every.ValueString() < intervals[j].Every.ValueString()
		}
		return intervals[i].Offset.ValueString() < intervals[j].Offset.ValueString()
	})

	inputPayload := basetypes.StringValue{}
	if len(actionDetails.GetInput().GetPayloads()) > 0 {
		inputPayload = types.StringValue(string(actionDetails.GetInput().GetPayloads()[0].GetData()))
	}

	return &scheduleResourceModel{
		Name:           types.StringValue(name),
		IsPaused:       types.BoolValue(response.GetSchedule().GetState().GetPaused()),
		PauseOnFailure: types.BoolValue(response.GetSchedule().GetPolicies().GetPauseOnFailure()),
		Action: scheduleActionModel{
			InputPayload:  inputPayload,
			WorkflowId:    types.StringValue(actionDetails.GetWorkflowId()),
			WorkflowType:  types.StringValue(actionDetails.GetWorkflowType().GetName()),
			TaskQueueName: types.StringValue(actionDetails.GetTaskQueue().GetName()),
		},
		Spec: scheduleSpecModel{
			Intervals: intervals,
		},
		OverlapPolicy: types.StringValue(scheduleOverlapPolicyToString(response.GetSchedule().Policies.OverlapPolicy)),
		CatchupWindow: types.StringValue(formatDuration(response.GetSchedule().GetPolicies().CatchupWindow.AsDuration())),
	}
}

func (r *scheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	opts, ok := req.ProviderData.(*providerConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *provider.providerConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = opts.client
	r.namespace = opts.namespace
}

// Metadata returns the resource type name.
func (r *scheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

// Schema defines the schema for the resource.
func (r *scheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The `temporal_schedule` resource allows you to create and manage schedules for Temporal workflows. A schedule in Temporal defines when and how frequently a workflow should be executed.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Schedule name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Required: true,
			},
			"is_paused": schema.BoolAttribute{
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether that schedule is currently paused.",
				Optional:    true,
			},
			"pause_on_failure": schema.BoolAttribute{
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether that schedule should be paused after a failure.",
				Optional:    true,
			},
			"overlap_policy": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Controls what happens when an Action would be started by a Schedule at the same time that an older Action is still running. One of: `skip`, `buffer_one`, `buffer_all`, `cancel_other`, `terminate_other`, `allow_all`.",
				Optional:            true,
				Default:             stringdefault.StaticString("skip"),
				Validators: []validator.String{
					validators.StringInSliceValidator{
						AllowedValues: []string{"skip", "buffer_one", "buffer_all", "cancel_other", "terminate_other", "allow_all"},
					},
				},
			},
			"catchup_window": schema.StringAttribute{
				Computed:    true,
				Description: "The Temporal Server might be down or unavailable at the time when a Schedule should take an Action. When the Server comes back up, CatchupWindow controls which missed Actions should be taken at that point. An outage that lasts longer than the Catchup Window could lead to missed Actions. E.g. \"10m\", \"3h\".",
				Default:     stringdefault.StaticString("365d"),
				Optional:    true,
				Validators: []validator.String{
					validators.StringDurationValidator{},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"action": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"workflow_id": schema.StringAttribute{
						Description: "ID given to the workflow execution this schedule starts. This is auto-generated by Temporal.",
						Computed:    true,
						Optional:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"workflow_type": schema.StringAttribute{
						Description: "Name of the workflow definition this schedule starts.",
						Required:    true,
					},
					"task_queue_name": schema.StringAttribute{
						Description: "Name of the queue in which the workflow execution will be placed.",
						Required:    true,
					},
					"input_payload": schema.StringAttribute{
						Description: "Input payload passed to the workflow execution. Must be a valid JSON string.",
						Optional:    true,
					},
				},
				Description: "Details about the action this schedule triggers.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Object{
					objectvalidator.IsRequired(),
				},
			},
			"spec": schema.SingleNestedBlock{
				Description: "Describes when a schedules action should occur.",
				Validators: []validator.Object{
					objectvalidator.IsRequired(),
				},
				Blocks: map[string]schema.Block{
					"interval": schema.SetNestedBlock{
						Description: "Interval-based specifications of times. Allows to run a workflow \"every X seconds|minutes|hours|days\".",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"every": schema.StringAttribute{
									Required:    true,
									Description: "Period to repeat the interval. E.g \"30s\", \"10m\", \"4h\", \"7d\".",
									Validators: []validator.String{
										validators.StringDurationValidator{},
									},
								},
								"offset": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Default:             stringdefault.StaticString("0s"),
									MarkdownDescription: "Fixed offset added to the intervals period. For example, an `every` of 1h with `offset` of 0s would match every hour, on the hour. The same `every` but an `offset` of 19m would match every `xx:19:00`.",
									Validators: []validator.String{
										validators.StringDurationValidator{},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *scheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *scheduleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var args = make([]interface{}, 0)
	if !data.Action.InputPayload.IsNull() {
		var d map[string]interface{}
		argsString := data.Action.InputPayload.ValueString()
		err := json.Unmarshal([]byte(argsString), &d)
		if err != nil {
			resp.Diagnostics.AddError("Invalid input_payload", err.Error())
			return
		}
		args = []interface{}{d}
	}

	intervals := make([]temporal.ScheduleIntervalSpec, 0)
	for _, i := range data.Spec.Intervals {
		every, err := parseDuration(i.Every.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error parsing schedule interval value", i.Every.ValueString())
			continue
		}
		var offset time.Duration
		if i.Offset.ValueString() != "" {
			offset, err = parseDuration(i.Offset.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error parsing schedule offset value", i.Offset.ValueString())
				continue
			}
		}
		intervals = append(intervals, temporal.ScheduleIntervalSpec{
			Every:  every,
			Offset: offset,
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	catchupWindow, err := parseDuration(data.CatchupWindow.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing the catchup window as a duration", data.CatchupWindow.ValueString())
		return
	}

	// Create the schedule action
	action := &temporal.ScheduleWorkflowAction{
		Workflow:  data.Action.WorkflowType.ValueString(),
		TaskQueue: data.Action.TaskQueueName.ValueString(),
		Args:      args,
	}

	// Only set the workflow ID if it's provided in the configuration
	if !data.Action.WorkflowId.IsNull() && !data.Action.WorkflowId.IsUnknown() {
		action.ID = data.Action.WorkflowId.ValueString()
	}

	_, err = r.client.ScheduleClient().Create(ctx, temporal.ScheduleOptions{
		ID: data.Name.ValueString(),
		Spec: temporal.ScheduleSpec{
			Intervals: intervals,
		},
		Action:         action,
		Overlap:        stringToScheduleOverlapPolicy(data.OverlapPolicy.ValueString()),
		CatchupWindow:  catchupWindow,
		PauseOnFailure: data.PauseOnFailure.ValueBool(),
		Paused:         data.IsPaused.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating schedule", err.Error())
		return
	}

	// After creating the schedule, we need to fetch it to get the computed values
	// but preserve the original workflow_id since Temporal uses it as a template
	schedule, err := r.client.WorkflowService().DescribeSchedule(ctx, &workflowservice.DescribeScheduleRequest{
		Namespace:  r.namespace,
		ScheduleId: data.Name.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError("Unable to fetch the Temporal schedule after creation.", err.Error())
		return
	}

	// Parse the response - always use the value returned from Temporal as source of truth
	parsedData := parseScheduleResource(data.Name.ValueString(), schedule)

	// If workflow_id was explicitly provided in the configuration, preserve it
	// Otherwise, use the auto-generated one from Temporal
	if !data.Action.WorkflowId.IsNull() && !data.Action.WorkflowId.IsUnknown() {
		parsedData.Action.WorkflowId = data.Action.WorkflowId
	}
	// If workflow_id was not provided, parsedData.Action.WorkflowId will contain the auto-generated one

	data = parsedData

	diags := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *scheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string

	diags := req.State.GetAttribute(ctx, path.Root("name"), &name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, err := r.client.WorkflowService().DescribeSchedule(ctx, &workflowservice.DescribeScheduleRequest{
		Namespace:  r.namespace,
		ScheduleId: name,
	})

	if err != nil {
		if err.Error() == "schedule not found" {
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError("Error fetching the Schedule "+name, err.Error())
		}
		return
	}

	data := parseScheduleResource(name, schedule)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *scheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var name string
	diags := req.State.GetAttribute(ctx, path.Root("name"), &name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var data *scheduleResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	intervals := make([]temporal.ScheduleIntervalSpec, 0)
	for _, i := range data.Spec.Intervals {
		every, err := parseDuration(i.Every.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error parsing schedule interval value", i.Every.ValueString())
			continue
		}
		var offset time.Duration
		if i.Offset.ValueString() != "" {
			offset, err = parseDuration(i.Offset.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error parsing schedule offset value", i.Offset.ValueString())
				continue
			}
		}
		intervals = append(intervals, temporal.ScheduleIntervalSpec{
			Every:  every,
			Offset: offset,
		})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	catchupWindow, err := parseDuration(data.CatchupWindow.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing the catchup window as a duration", data.CatchupWindow.ValueString())
		return
	}

	handle := r.client.ScheduleClient().GetHandle(ctx, name)
	err = handle.Update(ctx, temporal.ScheduleUpdateOptions{
		DoUpdate: func(i temporal.ScheduleUpdateInput) (*temporal.ScheduleUpdate, error) {
			i.Description.Schedule.State.Paused = data.IsPaused.ValueBool()
			i.Description.Schedule.Policy.PauseOnFailure = data.PauseOnFailure.ValueBool()
			i.Description.Schedule.Policy.Overlap = stringToScheduleOverlapPolicy(data.OverlapPolicy.ValueString())
			i.Description.Schedule.Policy.CatchupWindow = catchupWindow
			i.Description.Schedule.Spec = &temporal.ScheduleSpec{
				Intervals: intervals,
			}
			return &temporal.ScheduleUpdate{
				Schedule: &i.Description.Schedule,
			}, nil
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating the Schedule "+name, err.Error())
		return
	}

	schedule, err := r.client.WorkflowService().DescribeSchedule(ctx, &workflowservice.DescribeScheduleRequest{
		Namespace:  r.namespace,
		ScheduleId: name,
	})

	if err != nil {
		resp.Diagnostics.AddError("Error fetching the Schedule "+name, err.Error())
		return
	}

	data = parseScheduleResource(name, schedule)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *scheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data scheduleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.WorkflowService().DeleteSchedule(ctx, &workflowservice.DeleteScheduleRequest{
		ScheduleId: data.Name.ValueString(),
		Namespace:  r.namespace,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error while deleting schedule "+data.Name.ValueString(), err.Error())
		return
	}
}

func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
