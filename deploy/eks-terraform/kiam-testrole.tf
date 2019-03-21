#
# Create the kiam test role
#
resource "aws_iam_role" "kiam_testrole" {
  name        = "eks-${var.cluster_name}-kiam-testrole"
  description = "A role to test kiam with."

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "AWS": "${aws_iam_role.kiam_intermediary.arn}"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

  tags = "${local.default_tags}"
}
