# kiam eks cluster setup

This is a simple example how i got kiam working with an eks cluster completly running on spot instances which assumes instance role instead of having them directly assigned. Please change the server / agent deployment to your needs. I added node taints and selectors because kiam needs a dedicated node which doesn't proxy requests to the server. I also like the idea with an dedicated instance to give less privileges to an node and have the setup more secure.

### Server setup

Create an iam role which gets assumed by the server container and allow the underlying node to assume this role. The role-arn need to be set in the kiam-server-configmap.yaml as value for the label `assume-role-arn`.
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::ACCOUNT-ID-WITHOUT-HYPHENS:role/node-role-name-where-kiam-server-runs",
          "arn:aws:iam::ACCOUNT-ID-WITHOUT-HYPHENS:root"
        ]
      },
      "Action": "sts:AssumeRole"
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

### agent setup
The agent daemonset doesn't need any changes and can be deployed as it it.

### roles setup

All Roles needs to be able to get assumed by the (assumed) role for the sample. The following shows the Trust Relationship json for an role which can be assigned to an pod.

````json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "AWS": [
          "arn:aws:iam::ACCOUNT-ID-WITHOUT-HYPHENS:root",
          "arn:aws:iam::ACCOUNT-ID-WITHOUT-HYPHENS:role/kiam-server-assume-role"
        ]
      },
      "Action": "sts:AssumeRole"
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
````