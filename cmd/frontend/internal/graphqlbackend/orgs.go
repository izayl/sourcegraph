package graphqlbackend

import (
	"context"

	"sourcegraph.com/sourcegraph/sourcegraph/pkg/backend"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/db"
)

func (r *siteResolver) Orgs(args *struct {
	connectionArgs
}) *orgConnectionResolver {
	return &orgConnectionResolver{
		connectionResolverCommon: newConnectionResolverCommon(args.connectionArgs),
	}
}

type orgConnectionResolver struct {
	connectionResolverCommon
}

func (r *orgConnectionResolver) Nodes(ctx context.Context) ([]*orgResolver, error) {
	// 🚨 SECURITY: Only site admins can list orgs.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return nil, err
	}

	orgsList, err := backend.Orgs.List(ctx)
	if err != nil {
		return nil, err
	}

	var l []*orgResolver
	for _, org := range orgsList {
		l = append(l, &orgResolver{
			org: org,
		})
	}
	return l, nil
}

func (r *orgConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	// 🚨 SECURITY: Only site admins can count orgs.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return 0, err
	}

	count, err := db.Orgs.Count(ctx)
	return int32(count), err
}
