# distant proxy for some delay during testing

variable "nginx_upstream" {
  description = "server IP or hostname"
}

job "you-listen-here" {
  # datacenters = ["dc1"]
  datacenters = ["dev-us-west-2"]
  group "nginx"{
    network {
      port "http" {
        # static = 8081
        to = 80
      }
    }
    service {
      name = "you-listen-here"
      port = "http"
      tags = [
        "owner=gulducat@github",
        "traefik.enable=true",
        "traefik.http.routers.you-listen-here.tls=true",
        "traefik.http.routers.you-listen-here.rule=Host(`you-listen-here.devhashi.app`)",
      ]
      check {
        name     = "alive"
        type     = "tcp"
        port     = "http"
        interval = "5s"
        timeout  = "2s"
      }
    }
    task "nginx" {
      driver = "docker"
      config {
        ports = ["http"]
        image = "nginx:1.21.6-alpine"
      }
      env {
        NGINX_ENVSUBST_TEMPLATE_DIR = "/local"
      }
      template {
        destination = "local/default.conf.template"
        data = <<-EOF
        upstream backend {
          server ${var.nginx_upstream};
        }
        server {
          listen       80;
          listen  [::]:80;
          server_name _;
          location / {
            proxy_pass http://backend;
            proxy_set_header Host $host;
          }
          location /websocket {
            proxy_pass http://backend;
            proxy_set_header Connection Upgrade;
            proxy_set_header Upgrade websocket;
            proxy_set_header Host $host;
          }
        }
        EOF
      }
    }
  }
}
