resource "template_dir" "terraform_variables" {
  source_dir      = "${path.module}/manifests"
  destination_dir = "${path.module}/manifests_rendered"

  vars {
    terraform_kiam_server_target_role_arn   = "${aws_iam_role.kiam_intermediary.arn}"
    terraform_kiam_testrole_arn             = "${aws_iam_role.kiam_testrole.arn}"
    terraform_node_iam_role_arn             = "${aws_iam_role.eks_node.arn}"
    terraform_node_iam_role_kiam_server_arn = "${aws_iam_role.kiam_server_instance.arn}"
  }
}
