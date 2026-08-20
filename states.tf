locals {
  generateDates           = <<-EOT
    (
        $fmt:='[Y]-[M01]-[D01]';
        $today:=$data_movimento;
        [0..20].(
            $padHour:=$substring('0' & $, -2);
            $dt:=$toMillis($today&'T'&$padHour&':05:00');
            [{
                'src_ptt': $dt ~> $fromMillis('y=[Y]/m=[M01]/d=[D01]/h=[H01]', '+0300'),
                'dst_ptt': $dt ~> $fromMillis('dt_mvt='&$fmt),
                'exec': $dt ~> $fromMillis($fmt&'T[H01]:[m]:[s]', '+0100')
            }]
        )
    )
    EOT
  generateDatesSingleLine = join("", [for s in split("\n", local.generateDates) : trimspace(s)])
}

resource "aws_sfn_activity" "approver" {
  name = "${var.project_name}-approver-worker"
}

resource "aws_sfn_state_machine" "this" {
  name     = "${var.project_name}-orchestrator"
  role_arn = aws_iam_role.sfn_role.arn

  definition = jsonencode({
    Comment       = "A simple minimal example of the Amazon States Language"
    QueryLanguage = "JSONata"
    StartAt       = "Parse Input Data"
    States = {
      "Parse Input Data" = {
        Type = "Pass"
        Assign = {
          data_movimento = "{% $now('[Y]-[M01]-[D01]', '-0300') %}"
        }
        Next = "Generate Next Processing Dates"
      }

      "Generate Next Processing Dates" = {
        Type = "Pass"
        Output = {
          executions = "{% ${local.generateDatesSingleLine} %}"
        }
        Next = "Proccess Data"
      }

      "Proccess Data" = {
        Type  = "Map"
        Items = "{% $states.input.executions %}"
        ItemProcessor = {
          ProcessorConfig = {
            Mode = "INLINE"
          }
          StartAt = "Wait For Start Datetime"
          States = {
            "Wait For Start Datetime" = {
              Type      = "Wait"
              Timestamp = "{% $states.input.exec & '-03:00' %}"
              Next      = "Init Hourly Processing"
            }
            "Init Hourly Processing" = {
              Type = "Pass"
              Output = {
                data_movimento = "{% $data_movimento %}"
                src_partition  = "{% $states.input.src_ptt %}"
                dst_partition  = "{% $states.input.dst_ptt %}"
                job_run_id     = "{% $uuid() %}"
              }
              End = true
            }
          }
        }
        Next = "Consolidate Results"
      }

      "Consolidate Results" = {
        Type = "Pass"
        Output = {
          processed_partition = "{% 'data_movimento=' & $data_movimento & '/' %}"
          job_run_id          = "{% $uuid() %}"
        }
        Next = "Send To Approver"
      }

      "Send To Approver" = {
        Type     = "Task"
        Resource = aws_sfn_activity.approver.id
        End      = true
      }
    }
  })
}
