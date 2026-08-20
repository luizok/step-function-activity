data "aws_iam_policy_document" "states_trust_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "sfn_role" {
  name               = "${var.project_name}-sfn-role"
  assume_role_policy = data.aws_iam_policy_document.states_trust_policy.json
}

data "aws_iam_policy_document" "sfn_policy" {
  statement {
    effect    = "Allow"
    actions   = ["s3:GetObject", "s3:ListBucket"]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "sfn_policy" {
  name        = "${var.project_name}-sfn-policy"
  description = "Allows read-only access to a specific S3 bucket"
  policy      = data.aws_iam_policy_document.sfn_policy.json
}

resource "aws_iam_role_policy_attachment" "sfn_attachment" {
  role       = aws_iam_role.sfn_role.name
  policy_arn = aws_iam_policy.sfn_policy.arn
}