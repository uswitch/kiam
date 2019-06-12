# MODIFY THIS
# Your locally configured aws profile
profile = "<your local aws profile>"
# The role to assume from that profile to build the infrastructure on aws
assumed_role = "arn:aws:iam::<account id>:role/<role name>"

# The region you want to build your infrastructure in
region = "eu-west-1"

# Your VPC id
vpc_id = "vpc-123"
# your private subnets
private_subnet_ids = ["subnet-123a", "subnet-123b", "subnet-123c"]
# your public subnets
public_subnet_ids = ["subnet-123d", "subnet-123e", "subnet-123f"]

# NO NEED TO MODIFY THIS
cluster_name = "test" # The name of the cluster
cluster_version = "1.12" # The kubernetes version of the cluster
extra_security_groups = [] # Additional security groups to attach to the eks nodes
