package example_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mocks "brook/mocks/modules/example"
	brookexample "brook/modules/example"
)

func TestCreateExample(t *testing.T) {
	t.Run("returns the created example", func(t *testing.T) {
		store := mocks.NewMockStore(t)
		want := &brookexample.Example{ID: "1", Name: "foo"}
		store.EXPECT().CreateExample(context.Background(), "foo").Return(want, nil)

		got, err := brookexample.NewDependencies(&brookexample.DependenciesConfig{Store: store}).
			CreateExample(context.Background(), "foo")

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("wraps store errors", func(t *testing.T) {
		store := mocks.NewMockStore(t)
		store.EXPECT().CreateExample(context.Background(), "foo").Return(nil, errors.New("boom"))

		_, err := brookexample.NewDependencies(&brookexample.DependenciesConfig{Store: store}).
			CreateExample(context.Background(), "foo")

		require.Error(t, err)
	})

	t.Run("rejects a reserved name without calling the store", func(t *testing.T) {
		store := mocks.NewMockStore(t)

		_, err := brookexample.NewDependencies(&brookexample.DependenciesConfig{Store: store}).
			CreateExample(context.Background(), "admin")

		require.ErrorIs(t, err, brookexample.ErrReservedName)
	})
}
