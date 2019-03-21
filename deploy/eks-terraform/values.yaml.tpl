terraform_variables:
  kiam:
    server_target_role_arn: ${terraform_kiam_server_target_role_arn}
  general:
    eks_cluster_name: ${terraform_eks_cluster_name}
    eks_cluster_endpoint: ${terraform_eks_cluster_endpoint}
    vpc_id: ${terraform_vpc_id}
    region: ${terraform_region}
    node_iam_role_arn: ${terraform_node_iam_role_arn}
    node_iam_role_kiam_server_arn: ${terraform_node_iam_role_kiam_server_arn}
