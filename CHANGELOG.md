# Changelog

## [0.7.0](https://github.com/MooseTheRebel/ohpygossh/compare/ohpygossh-v0.6.2...ohpygossh-v0.7.0) (2026-09-15)


### Features

* bump all infra & configuration to go 1.26 ([#101](https://github.com/MooseTheRebel/ohpygossh/issues/101)) ([3b1cd06](https://github.com/MooseTheRebel/ohpygossh/commit/3b1cd060e36319063e2ea6f32c55d56838d1c18a))
* support Python 3.14 ([#113](https://github.com/MooseTheRebel/ohpygossh/issues/113)) ([327945d](https://github.com/MooseTheRebel/ohpygossh/commit/327945de3f6bf14d2d86bdd2fa1d5a7bdd036478))


### Bug Fixes

* build Python 3.13 wheels in CI ([#103](https://github.com/MooseTheRebel/ohpygossh/issues/103)) ([069264a](https://github.com/MooseTheRebel/ohpygossh/commit/069264ae02a6191910eefb363812a6905bdc96ea))
* duplicate build logic in setup.py & setup_ci.py ([a807341](https://github.com/MooseTheRebel/ohpygossh/commit/a8073416049852544474e64b13c0d966a5a3beb7))
* Go and golangci-lint version drift across CI ([#95](https://github.com/MooseTheRebel/ohpygossh/issues/95)) ([5b64291](https://github.com/MooseTheRebel/ohpygossh/commit/5b642914b0ef6ce020bb62e4e4c23e2b4a131a3f))
* remove .python-version file ([c58679f](https://github.com/MooseTheRebel/ohpygossh/commit/c58679fe7f3fbc6ebd30b479e34e56383fd2931c))
* remove .python-version file ([474683d](https://github.com/MooseTheRebel/ohpygossh/commit/474683dca7d1309dc20e7f0acb033081d1caca0f))
* remove duplicate shell scripts ([#107](https://github.com/MooseTheRebel/ohpygossh/issues/107)) ([b291003](https://github.com/MooseTheRebel/ohpygossh/commit/b291003ca1297f333e946e589b7f1799034aad47))
* unify CI workflows, dedupe wheel build steps ([#109](https://github.com/MooseTheRebel/ohpygossh/issues/109)) ([c9055b5](https://github.com/MooseTheRebel/ohpygossh/commit/c9055b5d3638a741dfa8b3d0395793747b2ca599))


### Miscellaneous Chores

* bump golang.org/x/crypto from 0.54.0 to 0.55.0 ([#91](https://github.com/MooseTheRebel/ohpygossh/issues/91)) ([a0ef873](https://github.com/MooseTheRebel/ohpygossh/commit/a0ef873611442504e470b783c15fd4009985e257))
* bump golang.org/x/crypto from 0.55.0 to 0.57.0 ([#96](https://github.com/MooseTheRebel/ohpygossh/issues/96)) ([e100753](https://github.com/MooseTheRebel/ohpygossh/commit/e100753fa5e40e774140b2d3784b7ddc4fdc735b))
* bump pypa/cibuildwheel from 4.1.1 to 4.2.1 ([#97](https://github.com/MooseTheRebel/ohpygossh/issues/97)) ([34cfca5](https://github.com/MooseTheRebel/ohpygossh/commit/34cfca5b5683abcb8e3f807c104d6b0a24766014))
* bump wheel from 0.47.0 to 0.48.0 ([#92](https://github.com/MooseTheRebel/ohpygossh/issues/92)) ([4eed647](https://github.com/MooseTheRebel/ohpygossh/commit/4eed6472290813a6d2cd6a0dcb503fa6b92ae795))

## [0.6.2](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.6.1...ohpygossh-v0.6.2) (2026-08-01)


### Miscellaneous Chores

* pin gopy version via go.mod tool directive ([#86](https://github.com/b-long/ohpygossh/issues/86)) ([cdd8c69](https://github.com/b-long/ohpygossh/commit/cdd8c69fa051f3cb2f3b9dd50694cad1285d1fc1))
* run 'just gopy-version-bump' ([#89](https://github.com/b-long/ohpygossh/issues/89)) ([d39d7ee](https://github.com/b-long/ohpygossh/commit/d39d7ee2b14bebecf0e15838b2de777bfa336fd8))

## [0.6.1](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.6.0...ohpygossh-v0.6.1) (2026-08-01)


### Bug Fixes

* chores should be visible in the CHANGELOG ([#82](https://github.com/b-long/ohpygossh/issues/82)) ([f555d05](https://github.com/b-long/ohpygossh/commit/f555d0508472610d591270a34f0b3139227ab240))


### Miscellaneous Chores

* bump pypa/cibuildwheel from 4.1.0 to 4.1.1 ([#84](https://github.com/b-long/ohpygossh/issues/84)) ([0db9a45](https://github.com/b-long/ohpygossh/commit/0db9a4569544c03ead52ed007347aa1adc993a67))

## [0.6.0](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.5.0...ohpygossh-v0.6.0) (2026-07-28)


### Features
* improve Developer Experience ([#77](https://github.com/b-long/ohpygossh/issues/77)) ([62e36c7](https://github.com/b-long/ohpygossh/commit/62e36c763d631cc7d9b37abb52bd7aa52274012f))


### Miscellaneous Chores

* bump actions/checkout from 6 to 7 ([#70](https://github.com/b-long/ohpygossh/issues/70)) ([942574f](https://github.com/b-long/ohpygossh/commit/942574ff53ddc7ef23ddda86267a1caef707879d))
* bump actions/setup-go from 5 to 7 ([#80](https://github.com/b-long/ohpygossh/issues/80)) ([b3aa5bd](https://github.com/b-long/ohpygossh/commit/b3aa5bd6ef11bb6651f5046ba6e9a89487a1ee61))
* bump actions/setup-python from 6 to 7 ([#79](https://github.com/b-long/ohpygossh/issues/79)) ([b8005ff](https://github.com/b-long/ohpygossh/commit/b8005ffaed6130c483a8e9f53f3b5471ff3e24cb))
* bump github.com/pkg/sftp from 1.13.10 to 1.13.11 ([#74](https://github.com/b-long/ohpygossh/issues/74)) ([bbfdbc0](https://github.com/b-long/ohpygossh/commit/bbfdbc034df0cdc82c2f6466c5a53ea799d3f1b8))
* bump golang.org/x/crypto from 0.45.0 to 0.54.0 ([#75](https://github.com/b-long/ohpygossh/issues/75)) ([67f1441](https://github.com/b-long/ohpygossh/commit/67f1441b93e02d4e4e326cfe8ce7231218193455))


## [0.5.0](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.4.0...ohpygossh-v0.5.0) (2026-07-09)


### Features

* support testing with LXD ([#68](https://github.com/b-long/ohpygossh/issues/68)) ([4995461](https://github.com/b-long/ohpygossh/commit/4995461e691df5971f8fb5dc4df7b92c6a17e651))
* update to go 1.25 ([#73](https://github.com/b-long/ohpygossh/issues/73)) ([055932b](https://github.com/b-long/ohpygossh/commit/055932b878995ad066bcce68bba6148e17742f41))
* update to latest gopy main (2026 June 02) ([#59](https://github.com/b-long/ohpygossh/issues/59)) ([d6c8cee](https://github.com/b-long/ohpygossh/commit/d6c8cee5643e575d11894f891453ba656194bbf3))

## [0.4.0](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.3.2...ohpygossh-v0.4.0) (2026-06-10)


### Features

* improve README for end-users ([#54](https://github.com/b-long/ohpygossh/issues/54)) ([120e815](https://github.com/b-long/ohpygossh/commit/120e815c9af6e42509ede0a85716f4a8d2f9abec))

## [0.3.2](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.3.1...ohpygossh-v0.3.2) (2026-05-29)


### Bug Fixes

* create new func for VM naming ([#51](https://github.com/b-long/ohpygossh/issues/51)) ([a1e8b86](https://github.com/b-long/ohpygossh/commit/a1e8b86c5923f4ea3ef8e0b7471bd76220437dc8))
* sync pyproject.toml to release-please manifest ([#50](https://github.com/b-long/ohpygossh/issues/50)) ([b7d6164](https://github.com/b-long/ohpygossh/commit/b7d61645f132338fb15ed3c616f59eae935e45e2))
* use explicit toml type in extra-files for pyproject.toml ([#52](https://github.com/b-long/ohpygossh/issues/52)) ([42b7084](https://github.com/b-long/ohpygossh/commit/42b70842a88a0d56627f44d5df720034b1813fb0))
* various cibuildwheel fixes ([#48](https://github.com/b-long/ohpygossh/issues/48)) ([338ee68](https://github.com/b-long/ohpygossh/commit/338ee68871eb6d834653c45ee2f367137be7d68b))

## [0.3.1](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.3.0...ohpygossh-v0.3.1) (2026-05-22)


### Bug Fixes

* add actions: write permission to dispatch publish workflow ([#31](https://github.com/b-long/ohpygossh/issues/31)) ([731e93a](https://github.com/b-long/ohpygossh/commit/731e93af4ac28770f5afbb67b7e763ee5778ce82))
* pass --repo to gh workflow run to avoid missing git context ([#30](https://github.com/b-long/ohpygossh/issues/30)) ([ccfb0c6](https://github.com/b-long/ohpygossh/commit/ccfb0c67f0e4a761909d88b20fdde39590c5858e))
* remove colima, docker, and vagrant steps from publish workflow ([#32](https://github.com/b-long/ohpygossh/issues/32)) ([fd6dee2](https://github.com/b-long/ohpygossh/commit/fd6dee25b348e2911f69ab94f5eb9278eb457e58))
* trigger publish via workflow_dispatch from release-please ([#28](https://github.com/b-long/ohpygossh/issues/28)) ([3b3e254](https://github.com/b-long/ohpygossh/commit/3b3e254a92c013c9624c540dc11226d701c4edfa))

## [0.3.0](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.2.0...ohpygossh-v0.3.0) (2026-05-22)


### Features

* add SFTP Upload and Download functions ([#26](https://github.com/b-long/ohpygossh/issues/26)) ([0cf669b](https://github.com/b-long/ohpygossh/commit/0cf669b0acef3424180e4c68a552c7591de4caaf))


### Bug Fixes

* PyPI publishing & tidy go-multipass tests ([#24](https://github.com/b-long/ohpygossh/issues/24)) ([5df9c8b](https://github.com/b-long/ohpygossh/commit/5df9c8bf47c83dad88b25d751cad5842de2c2bb5))
* sync pyproject.toml to 0.2.0, fix extra-files ([#27](https://github.com/b-long/ohpygossh/issues/27)) ([40ca820](https://github.com/b-long/ohpygossh/commit/40ca820adb0af974eabdedb8f56a63fbecc90bb7))

## [0.2.0](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.1.0...ohpygossh-v0.2.0) (2026-05-21)


### Features

* add multipass e2e test and CI workflows ([#23](https://github.com/b-long/ohpygossh/issues/23)) ([69b21d1](https://github.com/b-long/ohpygossh/commit/69b21d1c16b8edbd8285a008dff56a1f99cbc84c))
* expand go-lang tests ([#21](https://github.com/b-long/ohpygossh/issues/21)) ([7538ae5](https://github.com/b-long/ohpygossh/commit/7538ae5a9131ab9b82feacfef517eda9cbc10d63))

## [0.1.0](https://github.com/b-long/ohpygossh/compare/ohpygossh-v0.0.16...ohpygossh-v0.1.0) (2026-05-20)


### Bug Fixes

* drop TOML jsonpath & add version.txt  ([#19](https://github.com/b-long/ohpygossh/issues/19)) ([745513e](https://github.com/b-long/ohpygossh/commit/745513ebdbaf75f558a5bb67df03c1490a5de902))
* drop TOML jsonpath for pyproject.toml ([#18](https://github.com/b-long/ohpygossh/issues/18)) ([b1e8448](https://github.com/b-long/ohpygossh/commit/b1e844871131cd80e354e62c80fed6d50dee856c))
