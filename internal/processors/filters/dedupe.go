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

	hashes := make([]string, len(items))
	for i, item := range items {
		hashes[i] = hash.HashURL(item.GetLink())
	}

	existingList, err := store.ExistsByHash(ctx, hashes)
	if err != nil {
		logger.Error("dedupe: hash check failed, passing batch through", "processor", d.name, "error", err)
		return items, nil
	}

	existing := make(map[string]bool, len(existingList))
	for _, h := range existingList {
		existing[h] = true
	}

	out := make([]*types.Item, 0, len(items))
	seen := make(map[string]bool, len(items))
	for i, item := range items {
		h := hashes[i]
		if existing[h] || seen[h] {
			logger.Debug("dedupe: dropped item", "processor", d.name, "item_id", item.ID, "reason", "duplicate url")
			continue
		}
		seen[h] = true
		out = append(out, item)
	}
	return out, nil
}
