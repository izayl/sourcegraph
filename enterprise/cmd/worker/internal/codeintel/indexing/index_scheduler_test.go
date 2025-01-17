package indexing

import (
	"context"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/time/rate"

	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/schema"
)

func init() {
	indexSchedulerEnabled = func() bool { return true }
}

func TestIndexSchedulerUpdate(t *testing.T) {
	indexEnqueuer := NewMockIndexEnqueuer()

	mockDBStore := NewMockDBStore()
	mockDBStore.GetRepositoriesWithIndexConfigurationFunc.SetDefaultReturn([]int{43, 44, 45, 46}, nil)

	mockSettingStore := NewMockIndexingSettingStore()
	mockSettingStore.GetLastestSchemaSettingsFunc.SetDefaultReturn(&schema.Settings{
		SearchRepositoryGroups: map[string][]interface{}{},
	}, nil)

	mockRepoStore := NewMockIndexingRepoStore()
	mockRepoStore.ListRepoNamesFunc.SetDefaultReturn([]types.RepoName{
		{ID: 41}, {ID: 42}, {ID: 43},
	}, nil)

	scheduler := &IndexScheduler{
		dbStore:       mockDBStore,
		settingStore:  mockSettingStore,
		repoStore:     mockRepoStore,
		indexEnqueuer: indexEnqueuer,
		limiter:       rate.NewLimiter(25, 1),
		operations:    newOperations(&observation.TestContext),
	}

	if err := scheduler.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	if len(indexEnqueuer.QueueIndexesForRepositoryFunc.History()) != 6 {
		t.Errorf("unexpected number of calls to QueueIndexesForRepository. want=%d have=%d", 6, len(indexEnqueuer.QueueIndexesForRepositoryFunc.History()))
	} else {
		var repositoryIDs []int
		for _, call := range indexEnqueuer.QueueIndexesForRepositoryFunc.History() {
			repositoryIDs = append(repositoryIDs, call.Arg1)
		}
		sort.Ints(repositoryIDs)

		if diff := cmp.Diff([]int{41, 42, 43, 44, 45, 46}, repositoryIDs); diff != "" {
			t.Errorf("unexpected repository IDs (-want +got):\n%s", diff)
		}
	}
}

func TestDisabledAutoindexConfiguration(t *testing.T) {
	// ListRepoNames -> a, b, c, d
	// GetAutoindexDisabledRepositories -> c
	// Result: a, b, d
	indexEnqueuer := NewMockIndexEnqueuer()

	mockDBStore := NewMockDBStore()
	mockDBStore.GetRepositoriesWithIndexConfigurationFunc.SetDefaultReturn([]int{43, 44, 45, 46}, nil)
	mockDBStore.GetAutoindexDisabledRepositoriesFunc.SetDefaultReturn([]int{41, 50}, nil)

	mockSettingStore := NewMockIndexingSettingStore()
	mockSettingStore.GetLastestSchemaSettingsFunc.SetDefaultReturn(&schema.Settings{
		SearchRepositoryGroups: map[string][]interface{}{},
	}, nil)

	mockRepoStore := NewMockIndexingRepoStore()
	mockRepoStore.ListRepoNamesFunc.SetDefaultReturn([]types.RepoName{
		{ID: 41}, {ID: 42}, {ID: 43},
	}, nil)

	scheduler := &IndexScheduler{
		dbStore:       mockDBStore,
		settingStore:  mockSettingStore,
		repoStore:     mockRepoStore,
		indexEnqueuer: indexEnqueuer,
		limiter:       rate.NewLimiter(25, 1),
		operations:    newOperations(&observation.TestContext),
	}

	if err := scheduler.Handle(context.Background()); err != nil {
		t.Fatalf("unexpected error performing update: %s", err)
	}

	var repositoryIDs []int
	for _, call := range indexEnqueuer.QueueIndexesForRepositoryFunc.History() {
		repositoryIDs = append(repositoryIDs, call.Arg1)
	}
	sort.Ints(repositoryIDs)

	if diff := cmp.Diff([]int{42, 43, 44, 45, 46}, repositoryIDs); diff != "" {
		t.Errorf("unexpected repository IDs (-want +got):\n%s", diff)
	}
}
