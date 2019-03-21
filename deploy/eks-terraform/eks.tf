resource "aws_iam_role" "eks_cluster" {
  name        = "eks-${var.cluster_name}-service"
  description = "Allows Amazon EKS to manage your clusters on your behalf."

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "eks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]    
}
POLICY

  tags = "${local.default_tags}"
}

resource "aws_iam_role_policy_attachment" "eks_cluster_amazon_eks_cluster_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = "${aws_iam_role.eks_cluster.name}"
}

resource "aws_iam_role_policy_attachment" "eks_cluster_amazon_eks_service_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSServicePolicy"
  role       = "${aws_iam_role.eks_cluster.name}"
}

resource "aws_security_group" "eks_controlplane" {
  name        = "eks-ControlPlaneSecurityGroup-${var.cluster_name}"
  description = "Security group for the EKS control plane."
  vpc_id      = "${var.vpc_id}"

  tags = "${merge(
    local.default_tags,
    map(
      "Name", "EKS Control Plane Security Group (${var.cluster_name})",
    )
  )}"
}

resource "aws_security_group_rule" "eks_cluster_to_nodes_inbound" {
  description              = "Allow pods to communicate with the cluster API Server."
  type                     = "ingress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  security_group_id        = "${aws_security_group.eks_controlplane.id}"
  source_security_group_id = "${aws_security_group.eks_nodes.id}"
}

resource "aws_security_group_rule" "eks_cluster_to_nodes_outbound_1" {
  description              = "Allow the cluster control plane to communicate with pods running extension API servers on port 443."
  type                     = "egress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  security_group_id        = "${aws_security_group.eks_controlplane.id}"
  source_security_group_id = "${aws_security_group.eks_nodes.id}"
}

resource "aws_security_group_rule" "eks_cluster_to_nodes_outbound_2" {
  description              = "Allow the cluster control plane to communicate with worker Kubelet and pods."
  type                     = "egress"
  from_port                = 1025
  to_port                  = 65535
  protocol                 = "tcp"
  security_group_id        = "${aws_security_group.eks_controlplane.id}"
  source_security_group_id = "${aws_security_group.eks_nodes.id}"
}

resource "aws_eks_cluster" "cluster" {
  name     = "${var.cluster_name}"
  role_arn = "${aws_iam_role.eks_cluster.arn}"
  version  = "${var.cluster_version}"

  vpc_config {
    security_group_ids = ["${aws_security_group.eks_controlplane.id}"]
    subnet_ids         = ["${concat(var.private_subnet_ids, var.public_subnet_ids)}"]
  }

  depends_on = [
    "aws_iam_role_policy_attachment.eks_cluster_amazon_eks_cluster_policy",
    "aws_iam_role_policy_attachment.eks_cluster_amazon_eks_service_policy",
  ]
}
