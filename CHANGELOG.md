# Changelog

## [0.1.2](https://github.com/andrelair-platform/ktayl-policy-service/compare/ktayl-policy-service-v0.1.1...ktayl-policy-service-v0.1.2) (2026-08-17)


### Features

* **auth:** Authentik JWKS JWT middleware + scope-based authz (S007) ([#15](https://github.com/andrelair-platform/ktayl-policy-service/issues/15)) ([5d81952](https://github.com/andrelair-platform/ktayl-policy-service/commit/5d81952c979784007ebe00d62308bd00559702ae))
* CERT-1 M1-M2 sprint close — policy service complete ([190aa9a](https://github.com/andrelair-platform/ktayl-policy-service/commit/190aa9a1885d5a480269705263da71d4ce2de250))
* CERT-1 M1-M2 sprint close — policy service complete ([190aa9a](https://github.com/andrelair-platform/ktayl-policy-service/commit/190aa9a1885d5a480269705263da71d4ce2de250))
* **S003:** policy REST API — CRUD + OpenAPI spec + cursor pagination ([#14](https://github.com/andrelair-platform/ktayl-policy-service/issues/14)) ([c2368e1](https://github.com/andrelair-platform/ktayl-policy-service/commit/c2368e16617ffb2e6af61a09839f7ee68d02937f))
* **s004:** state machine transitions + audit log ([32d35ba](https://github.com/andrelair-platform/ktayl-policy-service/commit/32d35bad3b0428c7564ce79b1cc9ef88ca9ce3d9))
* **s005:** document generation + MinIO presigned URLs ([6ee1d1c](https://github.com/andrelair-platform/ktayl-policy-service/commit/6ee1d1c1b66101d9b083f30bb371cb1fafd9bfb8))
* **s006:** NATS CloudEvents publisher + async policy transitions ([d9e8b54](https://github.com/andrelair-platform/ktayl-policy-service/commit/d9e8b544e8d73b3ee57670b838a861d642a63c02))
* **s008:** golangci-lint config + 70% coverage gate ([401279d](https://github.com/andrelair-platform/ktayl-policy-service/commit/401279d52fe4d335deb04a2081a6a64559539f64))
* **s009:** integration tests + testcontainers lifecycle coverage ([8962ee0](https://github.com/andrelair-platform/ktayl-policy-service/commit/8962ee002babf6257f2570078b31be2568bd86eb))

## [0.1.1](https://github.com/andrelair-platform/ktayl-policy-service/compare/ktayl-policy-service-v0.1.0...ktayl-policy-service-v0.1.1) (2026-08-14)


### Features

* **S001:** Go scaffold — chi router, healthz, Makefile, CI, Containerfile ([dcc71ef](https://github.com/andrelair-platform/ktayl-policy-service/commit/dcc71efe8964758fb2d7e5abfd7f483c5ab63c66))
* S002 + docs site + CVE fixes (dev → staging) ([#3](https://github.com/andrelair-platform/ktayl-policy-service/issues/3)) ([5569b8b](https://github.com/andrelair-platform/ktayl-policy-service/commit/5569b8b65677372fbd70ca3c7a7e0e86cb2711e4))
* S002 + docs site + CVE fixes (staging → main) ([#4](https://github.com/andrelair-platform/ktayl-policy-service/issues/4)) ([abb133e](https://github.com/andrelair-platform/ktayl-policy-service/commit/abb133e334f64156153d535ca1b61f18bd63c3e1))
* S002 + docs site + CVE fixes (staging → main) ([#4](https://github.com/andrelair-platform/ktayl-policy-service/issues/4)) ([#7](https://github.com/andrelair-platform/ktayl-policy-service/issues/7)) ([0ac0e06](https://github.com/andrelair-platform/ktayl-policy-service/commit/0ac0e06b93b1bcf52fb5df31c5cdbfa6ef96d35c))
* **S002:** domain model — Policy, Coverage, Premium + PostgreSQL repository ([fb61564](https://github.com/andrelair-platform/ktayl-policy-service/commit/fb6156400c01fc4ed02f482ca79ae24924105f9e))


### Bug Fixes

* **build:** go 1.23.0 minimum in go.mod — removes toolchain directive incompatible with golang:1.23-alpine ([f9acd3b](https://github.com/andrelair-platform/ktayl-policy-service/commit/f9acd3b0aaae7b429321aed94f2c6d32cec39f7e))
* **ci:** buildkitd insecure registry for Harbor self-signed CA ([64cba9d](https://github.com/andrelair-platform/ktayl-policy-service/commit/64cba9d86aa35d83970711dd9d77cb61454362d7))
* **lint:** drop revive exported rule — no-comments policy applies to exported symbols ([118573c](https://github.com/andrelair-platform/ktayl-policy-service/commit/118573ca0ba5320c1bc24b583df0acf1eb8e6fdb))
* **lint:** explicitly disable revive exported + package-comments rules ([114a2e7](https://github.com/andrelair-platform/ktayl-policy-service/commit/114a2e7bde1733b4bafac4419667d0f97eb51250))
* **lint:** golangci-lint v2 config schema — linters.settings + issues.exclusions.rules ([66466e1](https://github.com/andrelair-platform/ktayl-policy-service/commit/66466e132bb8e2f32c30c4eff95ac52e5989837a))
* **lint:** golangci-lint v2.12.2 — exclusions nested under linters, default: none ([0e999a2](https://github.com/andrelair-platform/ktayl-policy-service/commit/0e999a28da758af1d51f62672ff7a08ea177fe91))
* **lint:** remove deprecated RealIP middleware, add NewRouter doc comment ([a0f1710](https://github.com/andrelair-platform/ktayl-policy-service/commit/a0f1710e6fd64b27b8ceac3f45804fb17c6274cd))
* **security:** upgrade to Go 1.24 — fixes CVE-2025-68121 in crypto/tls (CRITICAL) ([a84f904](https://github.com/andrelair-platform/ktayl-policy-service/commit/a84f904b185a2d38e1f471ecb03a68a7cd6a77fb))

## Changelog

All notable changes to ktayl-policy-service are documented here.

This file is maintained by [release-please](https://github.com/googleapis/release-please).
