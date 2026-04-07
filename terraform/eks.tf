module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version

  vpc_id                   = module.vpc.vpc_id
  subnet_ids               = module.vpc.private_subnets
  control_plane_subnet_ids = module.vpc.private_subnets

  # Public access to the Kubernetes API server for local kubectl access
  cluster_endpoint_public_access = true

  # Default add-ons
  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
    }
    aws-ebs-csi-driver = {
      most_recent = true
    }
  }

  eks_managed_node_groups = {
    crypto_engine_nodes = {
      min_size     = 1
      max_size     = 2
      desired_size = 1

      instance_types = [var.instance_type]
      capacity_type  = "ON_DEMAND"

      # Attach the IAM policy required for persistent volume claims (Prometheus/Redis)
      iam_role_additional_policies = {
        AmazonEBSCSIDriverPolicy = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
      }
    }
  }

  # Allow the creator of the cluster (your IAM user) to have admin access to the cluster
  enable_cluster_creator_admin_permissions = true
}
