data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./tools/migrations/models",
    "--dialect", "mysql",
  ]
}
variable "mysql_user" {
  type    = string
  default = getenv("MYSQL_USER")
}

variable "mysql_password" {
  type    = string
  default = getenv("MYSQL_PASSWORD")
}

variable "db_host" {
  type    = string
  default = getenv("DB_HOST")
}

variable "db_port" {
  type    = string
  default = getenv("DB_PORT")
}

variable "mysql_database" {
  type    = string
  default = getenv("MYSQL_DATABASE")
}

env "gorm" {
  src = data.external_schema.gorm.url

  // マイグレーション適用先
  url = format(
    "mysql://%s:%s@%s:%s/%s?charset=utf8mb4&parseTime=true&loc=Local",
    var.mysql_user,
    urlescape(var.mysql_password),
    var.db_host,
    var.db_port,
    var.mysql_database,
  )

  // マイグレーションファイル生成に必要
  // https://atlasgo.io/concepts/dev-database
  dev = format(
    "mysql://%s:%s@%s:%s/%s?charset=utf8mb4&parseTime=true&loc=Local",
    var.mysql_user,
    urlescape(var.mysql_password),
    var.db_host,
    var.db_port,
    "atlas",
  )

  migration {
    dir = "file://tools/migrations/ddl"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}" // 拡張子のこと
    }
  }
}