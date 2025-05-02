env "local" {
  url     = env("DATABASE_URL")
  dev_url = env("DATABASE_URL")
  migration {
    dir = "file://migrations"   # ← 直接指定
  }
}

env "neon" {
  url = env("NEON_URL")
  migration {
    dir = "file://migrations"
  }
}
