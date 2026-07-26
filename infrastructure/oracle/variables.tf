variable "tenancy-ocid" {
  description = "OCID of the tenancy"
  sensitive   = true
  type        = string
}
variable "user-ocid" {
  description = "OCID of the user"
  sensitive   = true
  type        = string
}
variable "private-key-path" {
  description = "Path to the private key used to connect to OCI"
  sensitive   = true
  type        = string
}
variable "private-key-password" {
  description = "Optional passphrase for the private key"
  nullable    = true
  sensitive   = true
  type        = string
}
variable "fingerprint" {
  description = "Fingerprint of the key-pair being used"
  sensitive   = true
  type        = string
}
variable "region" {
  const       = true
  default     = "us-ashburn-1"
  description = "Region where you have your OCI tenancy"
  type        = string
}
variable "cidr-block" {
  const       = true
  default     = "10.0.1.0/24"
  description = "CIDR block range for the subnet"
  type        = string
}
variable "compute-shape" {
  const       = true
  default     = "VM.Standard.E2.1.Micro"
  description = "Template for compute resources to be allocated"
  type        = string
}
variable "source-image" {
  const       = true
  default     = "Canonical-Ubuntu-24.04-Minimal-2026.07.17-0"
  description = "Initial image to create the instance"
  type        = string
}
variable "public-key-path" {
  description = "Path to the public key for SSHing into the instance"
  sensitive   = true
  type        = string
}
