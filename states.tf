locals {
  generateDates           = <<-EOT
    (
        $fmt:='[Y]-[M01]-[D01]';
        $today:=$data_movimento;
        [0..23].(
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
        Type   = "Pass"
        Output = "{% ${local.generateDatesSingleLine} %}"
        End    = true
      }
    }
  })
}
