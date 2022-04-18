job "go-domain-checker" {
  type        = "service"
  region      = "[[.region]]"
  datacenters = ["[[.datacenter]]"]
  namespace   = "[[.namespace]]"

  constraint = {
    attribute = "${meta.env}"
    operator  = "="
    value     = "[[.env]]"
  }

  constraint = {
    attribute = "${meta.target.domain-checker}"
    operator  = "="
    value     = "1"
  }

  constraint = {
    operator  = "distinct_property"
    attribute = "${attr.unique.network.ip-address}"
    value     = "[[.max_per_host]]"
  }

  spread {
    attribute = "${node.unique.id}"
  }

  update {
    max_parallel      = 1
    healthy_deadline  = "10m"
    progress_deadline = "15m"

    auto_revert = true
  }

  group "go-domain-checker" {
    count = "[[.count]]"

    restart {
      interval = "1m"
      mode     = "delay"
    }

    task "go-domain-checker" {
      driver = "docker"

      config = {
        image = "registry.lucky-team.pro/luckyads/go.domain-checker:[[.tag]]"
        ports = ["http"]
      }

      env {
        ENV                     = "[[.env]]"
        REGION                  = "[[.app_region]]"
        LOGGER_LEVEL            = "[[.logger_level]]"
        HTTP_PORT               = "6060"
        TICKER_SSL_CHECKER      = "[[.ticker_ssl_checker]]"
        TICKER_EASYLIST_CHECKER = "[[.ticker_easylist_checker]]"
        ENABLE_EASYLIST         = "[[.enable_easylist]]"
      }

      vault {
        policies    = ["reader"]
        change_mode = "noop"
        env         = false
      }

      template {
        data = <<EOH
          {{with secret "secrets/go-domain-checker"}}
          POSTGRES_MAINDB_CONNECTION_STRING="{{.Data.data.postgres_maindb_connection_string}}"
          {{end}}
          EOH

        destination = "secrets/file.env"
        change_mode = "restart"
        env         = true
      }

      resources {
        cpu    = "[[.cpu_limit]]"
        memory = "[[.memory_limit]]"
      }
    }

    service {
      name         = "http-go-domain-checker"
      port         = "http"
      address_mode = "host"

      tags = [
        "green",
        "[[.tag]]",
        "[[.env]]",
        "[[.namespace]]",
      ]

      meta {
        protocol    = "http"
        environment = "[[.env]]"
        dc          = "${attr.consul.datacenter}"
      }

      check {
        type     = "http"
        port     = "http"
        interval = "5s"
        timeout  = "1s"
        path     = "/check"
      }
    }

    network {
      port "http" {
        to = 6060
      }
    }
  }
}
