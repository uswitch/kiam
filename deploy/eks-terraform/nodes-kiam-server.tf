#
# Create the instance profile of the Kiam server ec2 nodes
#
resource "aws_iam_role" "kiam_server_instance" {
  name        = "eks-${var.cluster_name}-kiam-server-node-instance"
  description = "Role the Kiam server instances use."

  assume_role_policy = <<EOF
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
EOF

  tags = "${local.default_tags}"
}

resource "aws_iam_role_policy_attachment" "eks_kiam_server_node_amazon_eks_worker_node_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = "${aws_iam_role.kiam_server_instance.name}"
}

resource "aws_iam_role_policy_attachment" "eks_kiam_server_node_amazon_eks_cni_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = "${aws_iam_role.kiam_server_instance.name}"
}

resource "aws_iam_role_policy_attachment" "eks_kiam_server_node_amazon_ec2_container_registry_readonly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = "${aws_iam_role.kiam_server_instance.name}"
}

# this role is allowed to assume the intermediary Kiam role
resource "aws_iam_role_policy" "kiam_server" {
  name = "eks-${var.cluster_name}-kiam-server-node-instance"
  role = "${aws_iam_role.kiam_server_instance.name}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "sts:AssumeRole"
      ],
      "Resource": "${aws_iam_role.kiam_intermediary.arn}"
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "kiam_server" {
  name = "kiam_server-${var.cluster_name}"
  role = "${aws_iam_role.kiam_server_instance.name}"
}

#
# Create ASG and Launch Configuration
#
locals {
  eks-kiam-server-node-userdata = <<USERDATA
#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh --apiserver-endpoint '${aws_eks_cluster.cluster.endpoint}' --b64-cluster-ca '${aws_eks_cluster.cluster.certificate_authority.0.data}' --kubelet-extra-args '--node-labels=kiam/nodetype=server,kubernetes.io/role=kiam-server --register-with-taints=kiam/nodetype=server:NoExecute' '${var.cluster_name}'
USERDATA
}

resource "aws_launch_configuration" "eks_node_kiam_server" {
  associate_public_ip_address = false
  iam_instance_profile        = "${aws_iam_instance_profile.kiam_server.name}"
  image_id                    = "${data.aws_ami.eks.id}"
  instance_type               = "t3.small"
  name_prefix                 = "eks-${var.cluster_name}-kiam-server-nodes-"
  security_groups             = ["${aws_security_group.eks_nodes.id}"]
  user_data_base64            = "${base64encode(local.eks-kiam-server-node-userdata)}"
  enable_monitoring           = false

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_autoscaling_group" "eks_kiam_server" {
  desired_capacity          = "2"
  launch_configuration      = "${aws_launch_configuration.eks_node_kiam_server.id}"
  max_size                  = 2
  min_size                  = 2
  wait_for_capacity_timeout = 0

  name                = "eks-${var.cluster_name}-kiam-server-nodes"
  vpc_zone_identifier = ["${var.private_subnet_ids}"]

  tag {
    key                 = "Origin"
    value               = "Terraform"
    propagate_at_launch = true
  }

  tag {
    key                 = "Name"
    value               = "eks-${var.cluster_name}-kiam-server-node"
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
