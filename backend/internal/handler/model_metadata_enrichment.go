package handler

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const modelMetadataCatalogContextKey = "model_metadata_catalog"

func enrichModelsMetadata(c *gin.Context, body []byte) []byte {
	value, ok := c.Get(modelMetadataCatalogContextKey)
	if !ok {
		return body
	}
	catalog, ok := value.(interface {
		Resolve(string) service.ModelMetadata
	})
	if !ok {
		return body
	}
	return enrichModelsMetadataWith(body, catalog.Resolve)
}

// Supplement missing descriptive fields only. Preserve existing upstream fields,
// prices, IDs, ordering and the already-filtered model membership verbatim.
func enrichModelsMetadataWith(body []byte, lookup func(string) service.ModelMetadata) []byte {
	var list map[string]json.RawMessage
	if json.Unmarshal(body, &list) != nil {
		return body
	}
	var models []map[string]json.RawMessage
	if json.Unmarshal(list["data"], &models) != nil {
		return body
	}
	for _, model := range models {
		var id string
		if json.Unmarshal(model["id"], &id) != nil {
			continue
		}
		metadata, err := json.Marshal(lookup(id))
		if err != nil {
			continue
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(metadata, &fields) != nil {
			continue
		}
		for key, value := range fields {
			if _, present := model[key]; !present {
				model[key] = value
			}
		}
	}
	data, err := json.Marshal(models)
	if err != nil {
		return body
	}
	list["data"] = data
	enriched, err := json.Marshal(list)
	if err != nil {
		return body
	}
	return enriched
}
