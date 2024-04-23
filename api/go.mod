module github.com/openstack-k8s-operators/dataplane-operator/api

go 1.20

require (
	github.com/openstack-k8s-operators/infra-operator/apis v0.3.1-0.20240419144952-326611519a8c
	github.com/openstack-k8s-operators/lib-common/modules/common v0.3.1-0.20240420115137-a02d94f5aa66
	github.com/openstack-k8s-operators/lib-common/modules/storage v0.3.1-0.20240420115137-a02d94f5aa66
	github.com/openstack-k8s-operators/openstack-baremetal-operator/api v0.3.1-0.20240422041901-293e48aceb9b
	k8s.io/api v0.28.9
	k8s.io/apimachinery v0.28.9
	sigs.k8s.io/controller-runtime v0.16.5
)

require (
	github.com/cert-manager/cert-manager v1.13.5
	github.com/go-playground/validator/v10 v10.19.0
	github.com/openstack-k8s-operators/openstack-operator/apis v0.0.0-20240422120541-8f652bde5abf
	golang.org/x/exp v0.0.0-20240409090435-93d18d7e34b8
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.2 // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-openapi/jsonpointer v0.20.2 // indirect
	github.com/go-openapi/jsonreference v0.20.4 // indirect
	github.com/go-openapi/swag v0.22.9 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gophercloud/gophercloud v1.11.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/metal3-io/baremetal-operator/apis v0.5.1 // indirect
	github.com/metal3-io/baremetal-operator/pkg/hardwareutils v0.4.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/openshift/api v3.9.0+incompatible // indirect
	github.com/openstack-k8s-operators/barbican-operator/api v0.0.0-20240422095355-066b5ce845c1 // indirect
	github.com/openstack-k8s-operators/cinder-operator/api v0.3.1-0.20240422110332-6ed2fef78115 // indirect
	github.com/openstack-k8s-operators/designate-operator/api v0.0.0-20240403153039-29d27af23767 // indirect
	github.com/openstack-k8s-operators/glance-operator/api v0.3.1-0.20240422132508-f70e0bce1bb6 // indirect
	github.com/openstack-k8s-operators/heat-operator/api v0.3.1-0.20240422125749-ff05088a9c5f // indirect
	github.com/openstack-k8s-operators/horizon-operator/api v0.3.1-0.20240422122457-4fad41f6b28f // indirect
	github.com/openstack-k8s-operators/ironic-operator/api v0.3.1-0.20240408054123-cb7b79a22b47 // indirect
	github.com/openstack-k8s-operators/keystone-operator/api v0.3.1-0.20240422083029-9546ece5eb4f // indirect
	github.com/openstack-k8s-operators/lib-common/modules/openstack v0.3.1-0.20240420115137-a02d94f5aa66 // indirect
	github.com/openstack-k8s-operators/manila-operator/api v0.3.1-0.20240422122211-bccd8acbdde6 // indirect
	github.com/openstack-k8s-operators/mariadb-operator/api v0.3.1-0.20240418060416-9de2d1f1915e // indirect
	github.com/openstack-k8s-operators/neutron-operator/api v0.3.1-0.20240422111921-f979f931e18c // indirect
	github.com/openstack-k8s-operators/nova-operator/api v0.3.1-0.20240422112427-13e4c8de493e // indirect
	github.com/openstack-k8s-operators/octavia-operator/api v0.3.1-0.20240419104752-ab112a2c09f3 // indirect
	github.com/openstack-k8s-operators/ovn-operator/api v0.3.1-0.20240422140910-e68a45de92f4 // indirect
	github.com/openstack-k8s-operators/placement-operator/api v0.3.1-0.20240422132507-fcac8d9e33fc // indirect
	github.com/openstack-k8s-operators/swift-operator/api v0.3.1-0.20240418150616-4d9e60def8ba // indirect
	github.com/openstack-k8s-operators/telemetry-operator/api v0.3.1-0.20240422130014-0607c4aa4a7b // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.18.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.46.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rabbitmq/cluster-operator/v2 v2.6.0 // indirect
	github.com/rhobs/obo-prometheus-operator/pkg/apis/monitoring v0.69.0-rhobs1 // indirect
	github.com/rhobs/observability-operator v0.0.28 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/oauth2 v0.16.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/term v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.28.9 // indirect
	k8s.io/client-go v0.28.9 // indirect
	k8s.io/component-base v0.28.9 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340 // indirect
	k8s.io/utils v0.0.0-20240310230437-4693a0247e57 // indirect
	sigs.k8s.io/gateway-api v0.8.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

// custom RabbitmqClusterSpecCore for OpenStackControlplane (v2.6.0_patches_tag)
replace github.com/rabbitmq/cluster-operator/v2 => github.com/openstack-k8s-operators/rabbitmq-cluster-operator/v2 v2.6.1-0.20240313124519-961a0ee8bf7f //allow-merging
