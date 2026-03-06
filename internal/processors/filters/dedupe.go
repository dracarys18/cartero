package filters

import (
	"context"

	"cartero/internal/types"
	"cartero/internal/utils/hash"
)

type DedupeProcessor struct {
	name string
}

func NewDedupeProcessor(name string) *DedupeProcessor {
	return &DedupeProcessor{name: name}
}

func (d *DedupeProcessor) Name() string {
	return d.name
}

func (d *DedupeProcessor) Initialize(_ context.Context, _ types.StateAccessor) error {
	return nil
}

func (d *DedupeProcessor) DependsOn() []string {
	return []string{}
}

func (d *DedupeProcessor) Process(ctx context.Context, st types.StateAccessor, item *types.Item) error {
	h := hash.HashURL(item.GetURL())
	store := st.GetStorage().Items()
	logger := st.GetLogger()

	exists, err := store.ExistsByHash(ctx, h)
	if err != nil {
		return err
	}

	if exists {
		logger.Info("DedupeProcessor rejected item", "processor", d.name, "item_id", item.ID, "reason", "duplicate url")
		return types.NewFilteredError(d.name, item.ID, "duplicate url").
			WithDetail("hash", h)
	}

	return nil
}
