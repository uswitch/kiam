# MODIFY THIS
# your locally configured aws profile
profile = ""
# the role to assume from that profile
assumed_role = ""

# the region you want to build your infrastructure in
region = ""

# your VPC id
vpc_id = "vpc-123"
# your private subnets
private_subnet_ids = ["subnet-123a", "subnet-123b", "subnet-123c"]
# your public subnets
public_subnet_ids = ["subnet-123d", "subnet-123e", "subnet-123f"]

# NO NEED TO MODIFY THIS
cluster_name = "test" # the name of the cluster
cluster_version = "1.11" # the kubernetes version of the cluster
extra_security_groups = [] # additional security groups to attach to the eks nodes
