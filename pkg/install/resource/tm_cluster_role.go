package resource

import (
	"context"

	v1 "k8s.io/api/rbac/v1"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/telepresenceio/telepresence/v2/pkg/install"
)

type tmClusterRole int

var TrafficManagerClusterRole Instance = tmClusterRole(0)

func (ri tmClusterRole) clusterRole() *kates.ClusterRole {
	cr := new(kates.ClusterRole)
	cr.TypeMeta = kates.TypeMeta{
		Kind:       "ClusterRole",
		APIVersion: "rbac.authorization.k8s.io/v1",
	}
	cr.ObjectMeta = kates.ObjectMeta{
		Name: install.ManagerAppName,
	}
	return cr
}

func (ri tmClusterRole) Create(ctx context.Context) error {
	cl := ri.clusterRole()
	cl.Rules = []v1.PolicyRule{
		{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{""},
			Resources: []string{"services"},
		},
	}
	return create(ctx, cl)
}

func (ri tmClusterRole) Exists(ctx context.Context) (bool, error) {
	return exists(ctx, ri.clusterRole())
}

func (ri tmClusterRole) Delete(ctx context.Context) error {
	return remove(ctx, ri.clusterRole())
}

func (ri tmClusterRole) Update(_ context.Context) error {
	// Noop
	return nil
}
