package example_test

import (
	"context"
	"errors"
	"testing"

	brookexample "brook/internal/example"
	examplemocks "brook/mocks/example"

	"go.uber.org/zap"
)

func TestCreateExampleUsesStoragePort(t *testing.T) {
	want := &brookexample.Example{ID: "id-1", Name: "name"}
	storage := examplemocks.NewMockStore(t)
	storage.EXPECT().CreateExample(context.Background(), "name").Return(want, nil)
	d := brookexample.NewDependenciesForTest(zap.NewNop(), storage)

	got, err := d.CreateExample(context.Background(), "name")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("CreateExample() = %#v, want %#v", got, want)
	}
}

func TestCreateExampleWrapsStorageError(t *testing.T) {
	storeErr := errors.New("storage unavailable")
	storage := examplemocks.NewMockStore(t)
	storage.EXPECT().CreateExample(context.Background(), "name").Return(nil, storeErr)
	d := brookexample.NewDependenciesForTest(zap.NewNop(), storage)

	_, err := d.CreateExample(context.Background(), "name")
	if !errors.Is(err, storeErr) {
		t.Fatalf("CreateExample() error = %v, want wrapped storage error", err)
	}
}
