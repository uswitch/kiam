#
# Create the instance profile of the Kiam server ec2 nodes
#
resource "aws_iam_role" "eks_node" {
  name        = "eks-${var.cluster_name}-node-instance"
  description = "Role the normal worker node instances use."

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
POLICY

  tags = "${local.default_tags}"
}

resource "aws_iam_role_policy_attachment" "eks_node_amazon_eks_worker_node_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = "${aws_iam_role.eks_node.name}"
}

resource "aws_iam_role_policy_attachment" "eks_node_amazon_eks_cni_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = "${aws_iam_role.eks_node.name}"
}

resource "aws_iam_role_policy_attachment" "eks_node_amazon_ec2_container_registry_readonly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = "${aws_iam_role.eks_node.name}"
}

resource "aws_iam_instance_profile" "eks_node" {
  name = "eks-node-${var.cluster_name}"
  role = "${aws_iam_role.eks_node.name}"
}

resource "aws_security_group" "eks_nodes" {
  name        = "eks-${var.cluster_name}-nodes"
  description = "Security group for all nodes in the cluster."
  vpc_id      = "${var.vpc_id}"

  tags = "${merge(
    local.default_tags,
    map(
      "Name", "EKS Nodes Security Group (${var.cluster_name})",
      "kubernetes.io/cluster/${var.cluster_name}", "owned",
    )
  )}"
}

resource "aws_security_group_rule" "eks_nodes_inbound_1" {
  description              = "Allow node to communicate with each other."
  type                     = "ingress"
  from_port                = -1
  to_port                  = -1
  protocol                 = -1
  security_group_id        = "${aws_security_group.eks_nodes.id}"
  source_security_group_id = "${aws_security_group.eks_nodes.id}"
}

resource "aws_security_group_rule" "eks_nodes_inbound_2" {
  description              = "Allow worker Kubelets and pods to receive communication from the cluster control plane."
  type                     = "ingress"
  from_port                = 1025
  to_port                  = 65535
  protocol                 = "tcp"
  security_group_id        = "${aws_security_group.eks_nodes.id}"
  source_security_group_id = "${aws_security_group.eks_controlplane.id}"
}

resource "aws_security_group_rule" "eks_nodes_inbound_3" {
  description              = "Allow pods running extension API servers on port 443 to receive communication from cluster control plane."
  type                     = "ingress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  security_group_id        = "${aws_security_group.eks_nodes.id}"
  source_security_group_id = "${aws_security_group.eks_controlplane.id}"
}

resource "aws_security_group_rule" "eks_nodes_outbound_1" {
  description       = "Allow outbound traffic."
  type              = "egress"
  from_port         = -1
  to_port           = -1
  protocol          = -1
  security_group_id = "${aws_security_group.eks_nodes.id}"
  cidr_blocks       = ["0.0.0.0/0"]
}

#
# Create ASG and Launch Configuration
#

# EKS currently documents this required userdata for EKS worker nodes to
# properly configure Kubernetes applications on the EC2 instance.
# We utilize a Terraform local here to simplify Base64 encoding this
# information into the AutoScaling Launch Configuration.
# More information: https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html

locals {
  eks_node_userdata = <<USERDATA
#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh --apiserver-endpoint '${aws_eks_cluster.cluster.endpoint}' --b64-cluster-ca '${aws_eks_cluster.cluster.certificate_authority.0.data}' --kubelet-extra-args '--node-labels=kubernetes.io/role=k8s-worker' '${var.cluster_name}'
USERDATA
}

data "aws_ami" "eks" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amazon-eks-node-1.11-*"]
  }

  owners = ["602401143452"]
}

resource "aws_launch_configuration" "eks_node" {
  associate_public_ip_address = false
  iam_instance_profile        = "${aws_iam_instance_profile.eks_node.name}"
  image_id                    = "${data.aws_ami.eks.id}"
  instance_type               = "m4.large"
  name_prefix                 = "eks-${var.cluster_name}-worker-nodes-"
  security_groups             = ["${concat(list(aws_security_group.eks_nodes.id), var.extra_security_groups)}"]
  user_data_base64            = "${base64encode(local.eks_node_userdata)}"
  enable_monitoring           = false

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_autoscaling_group" "eks_nodes" {
  desired_capacity          = "3"
  launch_configuration      = "${aws_launch_configuration.eks_node.id}"
  max_size                  = "3"
  min_size                  = "3"
  wait_for_capacity_timeout = 0

  name                = "eks-${var.cluster_name}-nodes"
  vpc_zone_identifier = ["${var.private_subnet_ids}"]

  tag {
    key                 = "Origin"
    value               = "Terraform"
    propagate_at_launch = true
  }

  tag {
    key                 = "Name"
    value               = "eks-${var.cluster_name}-node"
    propagate_at_launch = true
  }

  tag {
    key                 = "kubernetes.io/cluster/${var.cluster_name}"
    value               = "owned"
    propagate_at_launch = true
  }

  lifecycle {
    create_before_destroy = true
  }
}
