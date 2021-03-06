package schema

import (
	"context"

	"github.com/go-ggz/ggz/config"
	"github.com/go-ggz/ggz/helper"
	"github.com/go-ggz/ggz/model"
	"github.com/go-ggz/ggz/schema/cache/lru"
	"github.com/go-ggz/ggz/schema/cache/memory"

	"gopkg.in/nicksrandall/dataloader.v4"
)

var userLoader *dataloader.Loader

func userBatch(ctx context.Context, keys []interface{}) []*dataloader.Result {
	var results []*dataloader.Result
	id, _ := helper.GetCacheID(keys[0].(string))

	user, err := model.GetUserByID(id.(int64))

	results = append(results, &dataloader.Result{
		Data:  user,
		Error: err,
	})

	return results
}

func init() {
	var cache dataloader.Cache
	switch config.Cache.Driver {
	case "lru":
		cache = lru.NewEngine(config.Cache.Prefix)
	case "memory":
		cache = memory.NewEngine(config.Cache.Prefix, config.Cache.Expire)
	default:
		cache = dataloader.NewCache()
	}

	userLoader = dataloader.NewBatchedLoader(userBatch, dataloader.WithCache(cache))
}
