variable "APP_VERSION" {
  default = "v0.1.0-SNAPSHOT"
}
variable "ALPINE_VERSION" {
  default = "3.22.2"
}
variable "BTCD_VERSION" {
  default = "v0.25.0"
}
variable "LND_VERSION" {
  default = "v0.19.3-beta.rc2"
}
variable "GO_VERSION" {
  default = "1.25.4-alpine3.22"
}

target "backend" {
  context = "."
  dockerfile = "./backend/Dockerfile"
  target = "backend-scratch"
  args = {
	  COMPILATION = "static"
	  ALPINE_VERSION = "${ALPINE_VERSION}"
	  GO_VERSION = "${GO_VERSION}"
   }
  tags = ["LMare/lightning-playground-backend:${APP_VERSION}"]
}

target "backend-alpine" {
  context = "."
  dockerfile = "./backend/Dockerfile"
  target = "backend-alpine"
  args = {
	  COMPILATION = "dynamic"
	  ALPINE_VERSION = "${ALPINE_VERSION}"
	  GO_VERSION = "${GO_VERSION}"
  }
  tags = ["LMare/lightning-playground-backend:${APP_VERSION}-alpine-${ALPINE_VERSION}"]
}

target "frontend" {
  context = "."
  dockerfile = "./frontend/Dockerfile"
  args = { COMPILATION = "static" }
  tags = ["LMare/lightning-playground-frontend:${APP_VERSION}"]
}

target "frontend-alpine" {
  context = "."
  dockerfile = "./frontend/Dockerfile"
  args = { COMPILATION = "dynamic" }
  tags = ["LMare/lightning-playground-frontend:${APP_VERSION}-alpine-${ALPINE_VERSION}"]
}

target "btcd" {
  context = "https://github.com/btcsuite/btcd.git#$BTCD_VERSION"
  dockerfile = "Dockerfile"
  tags = ["btcsuite/btcd:${BTCD_VERSION}"]
}
