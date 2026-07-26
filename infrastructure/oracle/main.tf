terraform {
  required_providers {
    http = {
      source  = "hashicorp/http"
      version = "~> 3.6.0"
    }
    oci = {
      source  = "oracle/oci"
      version = "~> 8.24.0"
    }
  }
}

data "http" "local-public-ip" { url = "https://api.ipify.org" }
locals { local-public-ip = trimspace(data.http.local-public-ip.response_body) }
output "local-public-ip" {
  description = "Public-facing IP address of the local machine"
  type        = string
  value       = local.local-public-ip
}

provider "oci" {
  tenancy_ocid         = var.tenancy-ocid
  user_ocid            = var.user-ocid
  private_key_path     = var.private-key-path
  private_key_password = var.private-key-password
  fingerprint          = var.fingerprint
  region               = var.region
}

resource "oci_identity_compartment" "compartment" {
  description = "Compartment for Fire-Alarm"
  name        = "fire-alarm"
}

module "vcn" {
  source  = "oracle-terraform-modules/vcn/oci"
  version = "~> 4.0.0"

  compartment_id = oci_identity_compartment.compartment.id
  tenancy_id     = var.tenancy-ocid

  create_internet_gateway = true
}

resource "oci_core_security_list" "security-list" {
  compartment_id = oci_identity_compartment.compartment.id
  vcn_id         = module.vcn.vcn_id

  # ICMPv4
  ## Echo
  egress_security_rules {
    destination = "0.0.0.0/0"
    protocol    = "1"
    icmp_options {
      type = 0 # Reply
    }
  }
  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "1"
    icmp_options {
      type = 8 # Request
    }
  }

  ## Path MTU Discovery
  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "1"
    icmp_options {
      code = 4
      type = 3
    }
  }

  ## Time Exceeded
  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "1"
    icmp_options {
      type = 11
    }
  }

  ## Parameter Problem
  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "1"
    icmp_options {
      type = 12
    }
  }

  # TCP
  ## SSH
  egress_security_rules {
    destination = "${local.local-public-ip}/32"
    protocol    = "6"
    tcp_options {
      max = 22
      min = 22
    }
  }
  ingress_security_rules {
    source   = "${local.local-public-ip}/32"
    protocol = "6"
    tcp_options {
      max = 22
      min = 22
    }
  }

  ## HTTP
  egress_security_rules {
    destination = "0.0.0.0/0"
    protocol    = "6"
    tcp_options {
      max = 80
      min = 80
    }
  }
  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "6"
    tcp_options {
      max = 80
      min = 80
    }
  }

  ## HTTPS
  egress_security_rules {
    destination = "0.0.0.0/0"
    protocol    = "6"
    tcp_options {
      max = 443
      min = 443
    }
  }
  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "6"
    tcp_options {
      max = 443
      min = 443
    }
  }

  # UDP
  ## HTTPS/3
  ingress_security_rules {
    source   = "0.0.0.0/0"
    protocol = "17"
    udp_options {
      max = 443
      min = 443
    }
  }
}

resource "oci_core_subnet" "subnet" {
  compartment_id = oci_identity_compartment.compartment.id
  vcn_id         = module.vcn.vcn_id

  cidr_block        = var.cidr-block
  route_table_id    = module.vcn.ig_route_id
  security_list_ids = [oci_core_security_list.security-list.id]
}

data "oci_core_images" "images" {
  compartment_id = oci_identity_compartment.compartment.id

  display_name = var.source-image
  shape        = var.compute-shape
  state        = "AVAILABLE"
}
locals { image-id = one(data.oci_core_images.images.images).id }
check "image-found" {
  assert {
    condition     = local.image-id != null
    error_message = "Could not find desired image"
  }
}

data "oci_identity_availability_domains" "ads" { compartment_id = var.tenancy-ocid }
data "oci_core_shapes" "shapes" {
  compartment_id = oci_identity_compartment.compartment.id

  count               = length(data.oci_identity_availability_domains.ads.availability_domains)
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[count.index].name
  image_id            = local.image-id
  shape               = var.compute-shape
}

locals {
  ad-with-shape = one([
    for index, shapes in data.oci_core_shapes.shapes : index + 1
    if contains([for shape in shapes.shapes : shape.name], var.compute-shape)
  ])
}

check "shape-found" {
  assert {
    condition     = local.ad-with-shape != null
    error_message = "There are no availability domains with your desired compute shape"
  }
}

module "compute-instance" {
  source  = "oracle-terraform-modules/compute-instance/oci"
  version = "~> 2.4.1"

  compartment_ocid = oci_identity_compartment.compartment.id
  subnet_ocids     = [oci_core_subnet.subnet.id]
  source_ocid      = local.image-id

  ad_number       = local.ad-with-shape
  public_ip       = "EPHEMERAL"
  shape           = var.compute-shape
  ssh_public_keys = file(var.public-key-path)
}

output "remote-public-ip" {
  description = "Public IP address of the remote server"
  type        = string
  value       = one(module.compute-instance.public_ip)
}
