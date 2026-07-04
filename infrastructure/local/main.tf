terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 4.5.0"
    }
  }
}

provider "docker" {}

resource "docker_compose" "compose" {
  config_paths = [
    "${path.module}/../../compose.yaml"
  ]
}