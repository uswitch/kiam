# IAM Policy

Kiam has two policy implications:

1. Any nodes that run the Server process must have permissions to call `sts:AssumeRole`.
2. Any roles that Pods wish to assume must have policy which trusts the nodes running the Server process.

## Cluster Node Policy

Create an IAM role that will be assigned to the instances in your cluster with the following policy:

```json
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
```

## Application Roles

For any role which is to be assumed by a Pod you'll need to ensure it also has a trust policy
that permits nodes in the cluster to assume the role. This is referred to as the Trust Relationship
in the AWS Console, and the `assume_role_policy` in Terraform.

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
        "AWS": "arn:aws:iam::<account-id>:role/<cluster-node-role-name>"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

## Server IAM Role

A role (different from the clutser node EC2 instance role) can optionally be
provided for the Server to assume before interacting with the STS API (via
`--assume-role-arn`).

Using a separate Server IAM role is desirable in cases where the cluster node
instances and IAM roles are replaced frequently. The Trust Relationship requires
a fully qualified ARN of an existing role (no wildcards can be used), so reusing
a role separate from any one cluster allows Pods to move between clusters
without reconfiguring their IAM roles.

The cluster node EC2 instance role will need to have `sts:AssumeRole` permission
for this role and for there to be an entry in the Server role's trust policy.
Application roles will need to have a trust policy entry for this role, instead
of the cluster node role as noted above.
