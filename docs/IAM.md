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
        "AWS": "arn:aws:iam::<account-id>:role/<cluster-node-role-nae>"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```