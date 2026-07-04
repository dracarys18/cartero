package filters

import (
	"context"

	"cartero/internal/processors/names"
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

func (d *DedupeProcessor) DependsOn() []string {
	return []string{names.Blocklist}
}

func (d *DedupeProcessor) Process(ctx context.Context, st types.StateAccessor, items []*types.Item) ([]*types.Item, error) {
	store := st.GetStorage().Entries()
	logger := st.GetLogger()

	out := make([]*types.Item, 0, len(items))
	for _, item := range items {
		h := hash.HashURL(item.GetLink())

		exists, err := store.ExistsByHash(ctx, h)
		if err != nil {
			logger.Error("dedupe: hash check failed, dropping item", "processor", d.name, "item_id", item.ID, "error", err)
			continue
		}
		if exists {
			logger.Debug("dedupe: dropped item", "processor", d.name, "item_id", item.ID, "reason", "duplicate url")
			continue
		}

		out = append(out, item)
	}
	return out, nil
}
