variable "APP_VERSION" {
  default = "v0.3.1"
}
variable "BTCD_VERSION" {
  default = "v0.25.0"
}

variable "LND_TAG" {
  default = "v0.20.0-beta-custom"
}

variable "ALPINE_TAG" {
  default = "3.22.2"
}
variable "GO_TAG" {
  default = "1.25.4-alpine3.22"
}

target "common-app-args" {
	args = {
		ALPINE_TAG = "${ALPINE_TAG}"
		GO_TAG = "${GO_TAG}"
	}
}

group "compiler" {
	targets = ["sidecar-compiler", "api-compiler"]
}

group "app-scratch" {
	targets = ["api-scratch"]
}
group "app-alpine" {
	targets = ["api-alpine"]
}
group "api" {
	targets = ["api-alpine", "api-scratch"]
}
group "sidecars" {
	targets = ["sidecar-api", "sidecar-lnd", "sidecar-btcd"]
}
group "default" {
	targets = ["api", "btcd", "lnd", "sidecars"]
}


target "api-compiler" {
  context = "."
  dockerfile = "./api/Dockerfile"
  target = "compiler"
  inherits = ["common-app-args"]
  tags = ["api-compiler:latest"]
}


target "api-scratch" {
  context = "."
  dockerfile = "./api/Dockerfile"
  target = "api-scratch"
  inherits = ["common-app-args"]
  tags = ["LMare/lightning-playground-api:${APP_VERSION}"]
}

target "api-alpine" {
  context = "."
  dockerfile = "./api/Dockerfile"
  target = "api-alpine"
  inherits = ["common-app-args"]
  tags = ["LMare/lightning-playground-api:${APP_VERSION}-alpine-${ALPINE_TAG}"]
}

target "btcd" {
  context = "https://github.com/btcsuite/btcd.git#${BTCD_VERSION}"
  dockerfile = "Dockerfile"
  tags = ["btcsuite/btcd:${BTCD_VERSION}"]
}

target "lnd" {
  context = "https://github.com/LMare/lnd.git#feature/gRPC-alias-color"
  dockerfile = "Dockerfile"
  args = {
	  checkout = "feature/gRPC-alias-color"
	  git_url = "https://github.com/LMare/lnd"
  }
  tags = ["LMare/lnd:${LND_TAG}"]
}

target "sidecar-api" {
  context = "."
  dockerfile = "./sidecar/Dockerfile"
  args = { TYPE_SIDECAR = "api" }
  inherits = ["common-app-args"]
  tags = ["LMare/lightning-playground-sidecar-api:${APP_VERSION}"]
}

target "sidecar-compiler" {
  context = "."
  dockerfile = "./sidecar/Dockerfile"
  inherits = ["common-app-args"]
  target = "compiler"
  tags = ["sidecar-compiler:latest"]
}


target "sidecar-lnd" {
  context = "."
  dockerfile = "./sidecar/Dockerfile"
  args = { TYPE_SIDECAR = "lnd" }
  inherits = ["common-app-args"]
  tags = ["LMare/lightning-playground-sidecar-lnd:${APP_VERSION}"]
}


target "sidecar-btcd" {
  context = "."
  dockerfile = "./sidecar/Dockerfile"
  args = { TYPE_SIDECAR = "btcd" }
  inherits = ["common-app-args"]
  tags = ["LMare/lightning-playground-sidecar-btcd:${APP_VERSION}"]
}
