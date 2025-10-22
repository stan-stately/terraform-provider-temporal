// Runs a Workflow every day at 1am UTC.
resource "temporal_schedule" "example" {
  name             = "Example Schedule"
  is_paused        = false
  pause_on_failure = false
  overlap_policy   = "skip"
  catchup_window   = "3h"

  action {
    workflow_type   = "exampleWorkflow"
    task_queue_name = "example-task-queue"
    input_payload = jsonencode({
      myVar = "abc"
    })
  }

  spec {
    interval {
      every  = "1d"
      offset = "1h"
    }
  }
}
