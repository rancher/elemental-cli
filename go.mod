module github.com/rancher/elemental-cli

go 1.16

// until https://github.com/zloylos/grsync/pull/20 is merged we need to use our fork
replace github.com/zloylos/grsync v1.6.1 => github.com/rancher-sandbox/grsync v1.6.2-0.20220526080038-4032e9b0e97c

require (
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/Microsoft/hcsshim v0.9.2 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20220113124808-70ae35bab23f // indirect
	github.com/cavaliergopher/grab/v3 v3.0.1
	github.com/distribution/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.16+incompatible
	github.com/docker/go-units v0.4.0
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-containerregistry v0.10.0
	github.com/hashicorp/go-getter v1.5.11
	github.com/hashicorp/go-multierror v1.1.1
	github.com/ishidawataru/sctp v0.0.0-20210707070123-9a39160e9062 // indirect
	github.com/itchyny/gojq v0.12.6 // indirect
	github.com/jaypipes/ghw v0.9.1-0.20220511134554-dac2f19e1c76
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/joho/godotenv v1.4.0
	github.com/kevinburke/ssh_config v1.1.0 // indirect
	github.com/letsencrypt/boulder v0.0.0-20220331220046-b23ab962616e // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mudler/go-pluggable v0.0.0-20211206135551-9263b05c562e
	github.com/mudler/luet v0.0.0-20220526130937-264bf53fe7ab
	github.com/mudler/yip v0.0.0-20220321143540-2617d71ea02a
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	github.com/packethost/packngo v0.21.0 // indirect
	github.com/rancher-sandbox/gofilecache v0.0.0-20210330135715-becdeff5df15 // indirect
	github.com/sanity-io/litter v1.5.5
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sigstore/cosign v1.7.0
	github.com/sigstore/rekor v0.4.1-0.20220114213500-23f583409af3
	github.com/sigstore/sigstore v1.3.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.12.0
	github.com/twpayne/go-vfs v1.7.2
	github.com/vishvananda/netns v0.0.0-20211101163701-50045581ed74 // indirect
	github.com/xanzy/ssh-agent v0.3.1 // indirect
	github.com/zcalusic/sysinfo v0.0.0-20210905121133-6fa2f969a900 // indirect
	github.com/zloylos/grsync v1.6.1
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/mount-utils v0.23.0
	pault.ag/go/topsort v0.1.1 // indirect
)
