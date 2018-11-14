# IAM
This document provides more detail on how to setup your clusters and Kiam with Amazon's IAM.

## Cluster Policies
Kiam runs separate Agent and Server processes. The server is the only process
that needs to call `sts:AssumeRole` and can be placed on an isolated set of EC2
instances that don't run other user workloads. Agents should run on all user
workload instances and intercept requests to the metadata API. 

EC2 Instances running user workloads (and the Kiam agent) don't need any IAM
permissions aside from those needed by your installer (kops, kube-aws etc.) or
platform (EKS etc). In all situations your user workload nodes should have an
extremely reduced set of IAM permissions. 

Kiam is designed so that the EC2 instances running the Kiam server are the only
ones that need IAM policy to call `sts:AssumeRole`. 

### Configuring a Server role
Kiam's server includes a flag that can be used to specify an IAM Role that will
be assumed by the Server before it requests credentials. We recommend you use
this so that it's easier to form trust relationships between the roles that your
Pods will assume and this Server role. 

Using a separate Server IAM role is desirable in cases where the cluster node
instances and IAM roles are replaced frequently. The Trust Relationship requires
a fully qualified ARN of an existing role (no wildcards can be used), so reusing
a role separate from any one cluster allows Pods to move between clusters
without reconfiguring their IAM roles.

For the rest of this document, we'll use an example server role of
`kiam-server`, with a full ARN of `arn:aws:iam::123456789012:role/kiam-server`. 
This is the role that you'll specify against Kiam Server's `--assume-role-arn`
flag.

With this you'll need IAM policy which permits the EC2 instances to call
`sts:AssumeRole` for the `kiam-server` role. This will ensure the server process
can assume the server role. You'll also need IAM policy attached to the server
role that permits it to call `sts:AssumeRole`. This ensures that the Kiam Server
can request credentials for other roles. 

#### Server Node Policy
This is the example policy that will allow the EC2 instance that runs the Server
process can assume the server role. 

The example below is expressed using
[Terraform](https://www.terraform.io/) and should help explain how AWS IAM resources are
connected. 

```hcl
resource "aws_iam_role" "server_node" {
  name = "server_node"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { "Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}
    
resource "aws_iam_role_policy" "server_node" {
  name = "server_node"
  role = "${aws_iam_role.server_node.name}"
  policy = <<EOF
  {
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "sts:AssumeRole"
      ],
      "Resource": "arn:aws:iam:123456789012:role/kiam-server"
    }
    ]
  }
EOF
}
    
resource "aws_iam_instance_profile" "server_node" {
  name = "server_node"
  role = "${aws_iam_role.server_node.name}"
}


resource "aws_iam_role" "server_role" {
  name = "kiam-server"
  description = "Role the Kiam Server process assumes"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::123456789012:role/server_node"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "server_policy" {
  name = "kiam_server_policy"
  description = "Policy for the Kiam Server process"
  
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

resource "aws_iam_policy_attachment" "server_policy_attach" {
  name = "kiam-server-attachment"
  roles = ["${aws_iam_role.server_role.name}"]
  policy_arn = "${aws_iam_policy.server_policy.arn}"
}
```

## Application Roles

For any role which is to be assumed by a Pod you'll need to ensure it also has a
trust policy that permits nodes in the cluster to assume the role. This is
referred to as the Trust Relationship in the AWS Console, and the
`assume_role_policy` in Terraform. 

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::123456789012:role/kiam-server"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

The cluster node EC2 instance role will need to have `sts:AssumeRole` permission
for this role and for there to be an entry in the Server role's trust policy.
Application roles will need to have a trust policy entry for this role, instead
of the cluster node role as noted above.
