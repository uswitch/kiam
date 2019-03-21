#
# Create the intermediary Kiam role
#
resource "aws_iam_role" "kiam_intermediary" {
  name        = "eks-${var.cluster_name}-kiam-intermediary"
  description = "Role the Kiam server process assumes. If you want to use Kiam to assume any role R, THIS is the role that R must trust."

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "AWS": "${aws_iam_role.kiam_server_instance.arn}"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

  tags = "${local.default_tags}"
}

# The intermediary role is allowed to assume ALL roles
resource "aws_iam_policy" "kiam_intermediary" {
  name        = "eks-${var.cluster_name}-kiam-intermediary"
  description = "Policy for the Kiam intermediary role. Managed by Terraform."

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "sts:AssumeRole"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "kiam_intermediary" {
  role       = "${aws_iam_role.kiam_intermediary.name}"
  policy_arn = "${aws_iam_policy.kiam_intermediary.arn}"
}
