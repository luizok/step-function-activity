output "approver_activity_arn" {
  value = aws_sfn_activity.approver.arn
}

output "state_machine_arn" {
  value = aws_sfn_state_machine.this.arn
}
